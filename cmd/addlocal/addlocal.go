package addlocal

import (
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sagan/ptool/client"
	"github.com/sagan/ptool/cmd"
	"github.com/sagan/ptool/config"
	"github.com/sagan/ptool/site/tpl"
	"github.com/sagan/ptool/util"
	"github.com/sagan/ptool/util/torrentutil"
)

var command = &cobra.Command{
	Use:         "addlocal {client} {file.torrent}...",
	Annotations: map[string]string{"cobra-prompt-dynamic-suggestions": "addlocal"},
	Short:       "Add local torrents to client.",
	Long: `Add local torrents to client.
It's possible to use "*" wildcard in filename to match multiple torrents. e.g.: "*.torrent".
`,
	Args: cobra.MatchAll(cobra.MinimumNArgs(2), cobra.OnlyValidArgs),
	RunE: addlocal,
}

var (
	paused             = false
	skipCheck          = false
	renameAdded        = false
	deleteAdded        = false
	addCategoryAuto    = false
	sequentialDownload = false
	defaultSite        = ""
	rename             = ""
	addCategory        = ""
	addTags            = ""
	savePath           = ""
)

func init() {
	command.Flags().BoolVarP(&skipCheck, "skip-check", "", false, "Skip hash checking when adding torrents")
	command.Flags().BoolVarP(&renameAdded, "rename-added", "", false,
		"Rename successfully added torrents to .added extension")
	command.Flags().BoolVarP(&deleteAdded, "delete-added", "", false, "Delete successfully added torrents")
	command.Flags().BoolVarP(&paused, "add-paused", "", false, "Add torrents to client in paused state")
	command.Flags().BoolVarP(&addCategoryAuto, "add-category-auto", "", false,
		"Automatically set category of added torrent to corresponding sitename")
	command.Flags().BoolVarP(&sequentialDownload, "sequential-download", "", false,
		"(qbittorrent only) Enable sequential download")
	command.Flags().StringVarP(&savePath, "add-save-path", "", "", "Set save path of added torrents")
	command.Flags().StringVarP(&defaultSite, "site", "", "", "Set default site of torrents")
	command.Flags().StringVarP(&addCategory, "add-category", "", "", "Manually set category of added torrents")
	command.Flags().StringVarP(&rename, "rename", "", "", "Rename added torrent (for dev/test only)")
	command.Flags().StringVarP(&addTags, "add-tags", "", "", "Set tags of added torrent (comma-separated)")
	cmd.RootCmd.AddCommand(command)
	command2.Flags().AddFlagSet(command.Flags())
	cmd.RootCmd.AddCommand(command2)
}

func addlocal(cmd *cobra.Command, args []string) error {
	clientName := args[0]
	args = args[1:]
	if renameAdded && deleteAdded {
		return fmt.Errorf("--rename-added and --delete-added flags are NOT compatible")
	}
	clientInstance, err := client.CreateClient(clientName)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	errorCnt := int64(0)
	torrentFilenames := util.ParseFilenameArgs(args...)
	if rename != "" && len(torrentFilenames) > 1 {
		return fmt.Errorf("--rename flag can only be used with exact one torrent file arg")
	}
	option := &client.TorrentOption{
		Pause:              paused,
		SavePath:           savePath,
		SkipChecking:       skipCheck,
		SequentialDownload: sequentialDownload,
		Name:               rename,
	}
	var fixedTags []string
	if addTags != "" {
		fixedTags = strings.Split(addTags, ",")
	}
	cntAll := len(torrentFilenames)
	cntAdded := int64(0)
	sizeAdded := int64(0)

	for i, torrentFilename := range torrentFilenames {
		if strings.HasSuffix(torrentFilename, ".added") {
			log.Tracef("!torrent (%d/%d) %s: skipped", i+1, cntAll, torrentFilename)
			continue
		}
		var torrentContent []byte
		var err error
		if torrentFilename == "-" {
			torrentContent, err = io.ReadAll(os.Stdin)
		} else {
			torrentContent, err = os.ReadFile(torrentFilename)
		}
		if err != nil {
			fmt.Printf("✕torrent (%d/%d) %s: failed to read file (%v)\n", i+1, cntAll, torrentFilename, err)
			errorCnt++
			continue
		}
		tinfo, err := torrentutil.ParseTorrent(torrentContent, 99)
		if err != nil {
			fmt.Printf("✕torrent (%d/%d) %s: failed to parse torrent (%v)\n", i+1, cntAll, torrentFilename, err)
			errorCnt++
			continue
		}
		sitename, err := tpl.GuessSiteByTrackers(tinfo.Trackers, defaultSite)
		if err != nil {
			log.Warnf("Failed to find match site for %s by trackers: %v", torrentFilename, err)
		}
		if addCategoryAuto {
			if sitename != "" {
				option.Category = sitename
			} else if addCategory != "" {
				option.Category = addCategory
			} else {
				option.Category = "Others"
			}
		} else {
			option.Category = addCategory
		}
		option.Tags = []string{}
		if sitename != "" {
			option.Tags = append(option.Tags, client.GenerateTorrentTagFromSite(sitename))
			siteConfig := config.GetSiteConfig(sitename)
			if siteConfig.GlobalHnR {
				option.Tags = append(option.Tags, "_hr")
			}
		}
		option.Tags = append(option.Tags, fixedTags...)
		err = clientInstance.AddTorrent(torrentContent, option, nil)
		if err != nil {
			fmt.Printf("✕torrent (%d/%d) %s: failed to add to client (%v) // %s\n", i+1, cntAll, torrentFilename, err, tinfo.ContentPath)
			errorCnt++
			continue
		}
		if renameAdded {
			err := os.Rename(torrentFilename, torrentFilename+".added")
			if err != nil {
				log.Debugf("Failed to rename successfully added torrent %s to .added extension: %v // %s", torrentFilename, err, tinfo.ContentPath)
			}
		} else if deleteAdded {
			err := os.Remove(torrentFilename)
			if err != nil {
				log.Debugf("Failed to delete successfully added torrent %s: %v // %s", torrentFilename, err, tinfo.ContentPath)
			}
		}
		cntAdded++
		sizeAdded += tinfo.Size
		fmt.Printf("✓torrent (%d/%d) %s: added to client // %s\n", i+1, cntAll, torrentFilename, tinfo.ContentPath)
	}
	fmt.Printf("\nDone. Added torrent (Size/Cnt): %s / %d; ErrorCnt: %d\n", util.BytesSize(float64(sizeAdded)), cntAdded, errorCnt)
	if errorCnt > 0 {
		return fmt.Errorf("%d errors", errorCnt)
	}
	return nil
}

var command2 = &cobra.Command{
	Use:   "addlocal2 {client}",
	Short: `Alias of "addlocal --add-category-auto --delete-added {client} *.torrent"`,
	Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		addCategoryAuto = true
		deleteAdded = true
		args = append(args, "*.torrent")
		return command.RunE(cmd, args)
	},
}
