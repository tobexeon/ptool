package setcategory

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sagan/ptool/client"
	"github.com/sagan/ptool/cmd"
)

var command = &cobra.Command{
	Use:   "setcategory <client> <category> <infoHash>...",
	Short: "Set category of torrents in client",
	Long: `Set category of torrents in client
<infoHash>...: infoHash list of torrents. It's possible to use state filter to target multiple torrents:
_all, _active, _done,  _downloading, _seeding, _paused, _completed, _error`,
	Args: cobra.MatchAll(cobra.MinimumNArgs(3), cobra.OnlyValidArgs),
	Run:  createtags,
}

func init() {
	cmd.RootCmd.AddCommand(command)
}

func createtags(cmd *cobra.Command, args []string) {
	clientInstance, err := client.CreateClient(args[0])
	if err != nil {
		log.Fatal(err)
	}
	cat := args[1]
	args = args[2:]
	infoHashes, err := client.SelectTorrents(clientInstance, "", "", "", args...)
	if err != nil {
		log.Fatal(err)
	}
	if infoHashes == nil {
		err = clientInstance.SetAllTorrentsCatetory(cat)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err = clientInstance.SetTorrentsCatetory(infoHashes, cat)
		if err != nil {
			log.Fatal(err)
		}
	}
}
