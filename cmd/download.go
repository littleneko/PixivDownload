/*
Copyright © 2023 litao.little@gmail.com

*/

package cmd

import (
	"encoding/json"
	"math"
	"os"
	"os/signal"
	"pixiv/app"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download illust from pixiv",
	Long: `Download the illust from your bookmarks, all illust of your following,
all illust of the users id you specified, or from a illust id list.
You can run it as service mode by --service flag, it will check and download new illust periodically.`,
	Run: func(cmd *cobra.Command, args []string) {
		app.InitLog(viper.GetString("log-path"), viper.GetString("log-level"))

		var options app.PixivDlOptions
		err := viper.Unmarshal(&options)
		if err != nil {
			log.Fatalf("Failed to read config file, msg: %s", err)
		} else {
			j, _ := json.MarshalIndent(options, "", "  ")
			log.Infof("Use options: %s", string(j))
		}
		illustInfoManager, err := app.GetIllustInfoManager(&options)
		cobra.CheckErr(err)
		err = app.Start(&options, illustInfoManager)
		cobra.CheckErr(err)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
	},
}

func init() {
	downloadCmd.Flags().String("database-type", "SQLITE", "Database to store the illust info, 'NONE' means not use database and not check illust exist, choices: ['NONE', 'SQLITE']")
	downloadCmd.Flags().String("sqlite-path", "storage", "Sqlite file location if use sqlite database")
	downloadCmd.Flags().String("download-path", "pixiv", "Download file location")
	downloadCmd.Flags().String("filename-pattern", "{id}", "Filename pattern, all tag: ['user_id, 'user', 'id', 'title']")
	downloadCmd.Flags().Int32("scan-interval-sec", 3600, "The interval to check new illust if run in service mode")
	downloadCmd.Flags().Int32("parse-parallel", 5, "Parallel to get an parse illust info")
	downloadCmd.Flags().Int32("download-parallel", 10, "Parallel to download illust")
	downloadCmd.Flags().Int32("max-retries", math.MaxInt32, "Max retry times")
	downloadCmd.Flags().Int32("retry-backoff-ms", 1000, "Backoff time if request failed")
	downloadCmd.Flags().Int32("parse-timeout-ms", 5000, "Timeout for get illust info")
	downloadCmd.Flags().Int32("download-timeout-ms", 10000, "Timeout for download illust")
	downloadCmd.Flags().StringSlice("user-white-list", []string{}, "Only illust user id in this list will be download")
	downloadCmd.Flags().StringSlice("user-block-list", []string{}, "Illust user id in this list will skip to download")

	downloadCmd.Flags().String("cookie", "", "Your Cookies, only need the 'PHPSESSID=abcxyz'")
	downloadCmd.Flags().String("user-agent", "", "Http User-Agent header")

	downloadCmd.Flags().StringSlice("download-scope", []string{"ALL"}, "What to download, choices: ['ALL', 'BOOKMARKS', 'FOLLOWING', 'USER', 'ILLUST']")
	downloadCmd.Flags().String("user-id", "", "Download all bookmarks or following user's illust, if download-scope include bookmarks/following")
	downloadCmd.Flags().StringSlice("download-user-ids", []string{}, "Download all illust of this user")
	downloadCmd.Flags().StringSlice("download-illust-ids", []string{}, "Illust id to download")

	downloadCmd.Flags().Bool("no-r18", false, "Do not download R18 illust")
	downloadCmd.Flags().Bool("only-p0", false, "Only download the first picture of the illust if a multi picture illust")
}
