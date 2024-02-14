package add

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sagan/ptool/client"
	"github.com/sagan/ptool/cmd"
	"github.com/sagan/ptool/site"
	"github.com/sagan/ptool/site/tpl"
	"github.com/sagan/ptool/util"
	"github.com/sagan/ptool/util/torrentutil"
)

var command = &cobra.Command{
	Use:         "add {client} {torrentId | torrentUrl}...",
	Annotations: map[string]string{"cobra-prompt-dynamic-suggestions": "add"},
	Short:       "Add site torrents to client.",
	Long:        `Add site torrents to client.`,
	Args:        cobra.MatchAll(cobra.MinimumNArgs(2), cobra.OnlyValidArgs),
	RunE:        add,
}

var (
	addCategoryAuto    = false
	addPaused          = false
	skipCheck          = false
	sequentialDownload = false
	addCategory        = ""
	defaultSite        = ""
	addTags            = ""
	savePath           = ""
)

func init() {
	command.Flags().BoolVarP(&skipCheck, "skip-check", "", false, "Skip hash checking when adding torrents")
	command.Flags().BoolVarP(&addPaused, "add-paused", "", false, "Add torrents to client in paused state")
	command.Flags().BoolVarP(&addCategoryAuto, "add-category-auto", "", false,
		"Automatically set category of added torrent to corresponding sitename")
	command.Flags().BoolVarP(&sequentialDownload, "sequential-download", "", false,
		"(qbittorrent only) Enable sequential download")
	command.Flags().StringVarP(&addCategory, "add-category", "", "", "Set category of added torrents")
	command.Flags().StringVarP(&savePath, "add-save-path", "", "", "Set save path of added torrents")
	command.Flags().StringVarP(&defaultSite, "site", "", "", "Set default site of torrents")
	command.Flags().StringVarP(&addTags, "add-tags", "", "", "Add tags to added torrent (comma-separated)")
	cmd.RootCmd.AddCommand(command)
	command2.Flags().AddFlagSet(command.Flags())
	cmd.RootCmd.AddCommand(command2)
}

func add(cmd *cobra.Command, args []string) error {
	clientName := args[0]
	torrentIds := args[1:]
	clientInstance, err := client.CreateClient(clientName)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	domainSiteMap := map[string]string{}
	siteInstanceMap := map[string]site.Site{}
	errorCnt := int64(0)
	option := &client.TorrentOption{
		Pause:              addPaused,
		SavePath:           savePath,
		SkipChecking:       skipCheck,
		SequentialDownload: sequentialDownload,
	}
	var fixedTags []string
	if addTags != "" {
		fixedTags = strings.Split(addTags, ",")
	}

	for _, torrentId := range torrentIds {
		siteName := defaultSite
		if !util.IsUrl(torrentId) {
			i := strings.Index(torrentId, ".")
			if i != -1 && i < len(torrentId)-1 {
				siteName = torrentId[:i]
				torrentId = torrentId[i+1:]
			}
		} else {
			domain := util.GetUrlDomain(torrentId)
			if domain == "" {
				fmt.Printf("✕torrent %s: failed to parse domain", torrentId)
				errorCnt++
				continue
			}
			sitename := ""
			ok := false
			if sitename, ok = domainSiteMap[domain]; !ok {
				domainSiteMap[domain], err = tpl.GuessSiteByDomain(domain, defaultSite)
				if err != nil {
					log.Warnf("Failed to find match site for %s: %v", domain, err)
				}
				sitename = domainSiteMap[domain]
			}
			if sitename == "" {
				log.Warnf("Torrent %s: url does not match any site. will use provided default site", torrentId)
			} else {
				siteName = sitename
			}
		}
		if siteName == "" {
			fmt.Printf("✕torrent %s: no site found or provided\n", torrentId)
			errorCnt++
			continue
		}
		if siteInstanceMap[siteName] == nil {
			siteInstance, err := site.CreateSite(siteName)
			if err != nil {
				return fmt.Errorf("failed to create site %s: %v", siteName, err)
			}
			siteInstanceMap[siteName] = siteInstance
		}
		siteInstance := siteInstanceMap[siteName]
		torrentContent, _, err := siteInstance.DownloadTorrent(torrentId)
		if err != nil {
			fmt.Printf("✕add site %s torrent %s error: failed to get site torrent: %v\n", siteInstance.GetName(), torrentId, err)
			errorCnt++
			continue
		}
		tinfo, err := torrentutil.ParseTorrent(torrentContent, 99)
		if err != nil {
			fmt.Printf("✕add site %s torrent %s error: failed to parse torrent: %v\n", siteInstance.GetName(), torrentId, err)
			errorCnt++
			continue
		}
		if addCategoryAuto {
			if siteName != "" {
				option.Category = siteName
			} else if addCategory != "" {
				option.Category = addCategory
			} else {
				option.Category = "Others"
			}
		} else {
			option.Category = addCategory
		}
		option.Tags = []string{client.GenerateTorrentTagFromSite(siteName)}
		option.Tags = append(option.Tags, fixedTags...)
		if siteInstance.GetSiteConfig().GlobalHnR {
			option.Tags = append(option.Tags, "_hr")
		}
		err = clientInstance.AddTorrent(torrentContent, option, nil)
		if err != nil {
			fmt.Printf("✕add site %s torrent %s error: failed to add torrent to client: %v // %s\n", siteInstance.GetName(), torrentId, err, tinfo.ContentPath)
			errorCnt++
			continue
		}
		fmt.Printf("✓add site %s torrent %s success. infoHash=%s // %s\n", siteInstance.GetName(), torrentId, tinfo.InfoHash, tinfo.ContentPath)
	}
	if errorCnt > 0 {
		return fmt.Errorf("%d errors", errorCnt)
	}
	return nil
}

var command2 = &cobra.Command{
	Use:   "add2 [args]",
	Short: `Alias of "add --add-category-auto [args]"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addCategoryAuto = true
		return command.RunE(cmd, args)
	},
}
