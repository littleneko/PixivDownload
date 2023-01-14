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
You can run it as service mode by '--service-mode' flag, it will check
and download new illust periodically.`,
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

		if len(options.DownloadBookmarksUserIds) > 0 {
			downloader := app.NewBookmarksDownloader(&options, illustInfoManager)
			if options.ServiceMode {
				go downloader.Start()
			} else {
				downloader.Start()
				downloader.Close()
			}
		}
		if len(options.DownloadIllustIds) > 0 {
			downloader := app.NewIllustDownloader(&options, illustInfoManager)
			if options.ServiceMode {
				go downloader.Start()
			} else {
				downloader.Start()
				downloader.Close()
			}
		}

		if len(options.DownloadArtistUserIds) > 0 {
			downloader := app.NewArtistDownloader(&options, illustInfoManager)
			if options.ServiceMode {
				go downloader.Start()
			} else {
				downloader.Start()
				downloader.Close()
			}
		}

		if options.ServiceMode {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
		}
	},
}

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"

func init() {
	downloadCmd.Flags().Bool("service-mode", false, "Run as service mode, check and download new illust periodically")
	downloadCmd.Flags().String("database-type", "SQLITE", "Database to store the illust info, 'NONE' means not use database and not check illust exist, choices: ['NONE', 'SQLITE']")
	downloadCmd.Flags().String("sqlite-path", "storage", "Sqlite file location if use sqlite database")
	downloadCmd.Flags().String("download-path", "pixiv", "Download file location")
	downloadCmd.Flags().String("filename-pattern", "{id}", "Filename pattern, all tag can use: ['user_id, 'user', 'id', 'title']")
	downloadCmd.Flags().Int32("scan-interval-sec", 3600, "The interval to check new illust if run in service mode")
	downloadCmd.Flags().Int32("parse-parallel", 5, "Parallel number to get an parse illust info")
	downloadCmd.Flags().Int32("download-parallel", 10, "Parallel number to download illust")
	downloadCmd.Flags().Int32("max-retries", math.MaxInt32, "Max retry times")
	downloadCmd.Flags().Int32("retry-backoff-ms", 10000, "Backoff time if request failed")
	downloadCmd.Flags().Int32("parse-timeout-ms", 5000, "Timeout for get illust info")
	downloadCmd.Flags().Int32("download-timeout-ms", 600000, "Timeout for download illust")

	downloadCmd.Flags().StringSlice("download-bookmarks-uids", []string{}, "Download all bookmarks illust of this user")
	downloadCmd.Flags().StringSlice("download-following-uids", []string{}, "Download all following user's illust of this user")
	downloadCmd.Flags().StringSlice("download-artist-uids", []string{}, "Download all illust of this user")
	downloadCmd.Flags().StringSlice("download-illust-ids", []string{}, "Illust ids to download")

	downloadCmd.Flags().StringSlice("user-white-list", []string{}, "Only download illust which user id in this list")
	downloadCmd.Flags().StringSlice("user-block-list", []string{}, "Not download illust which user id in this list")

	downloadCmd.Flags().Bool("no-r18", false, "Not download R18 illust")
	downloadCmd.Flags().Bool("only-p0", false, "Only download the first picture of the illust if a multi picture illust it")
	downloadCmd.Flags().Int("bookmark-gt", -1, "Only download the illust bookmarks count great then it")
	downloadCmd.Flags().Int("like-gt", -1, "Only download the illust like count great then it")
	downloadCmd.Flags().Int("pixel-gt", -1, "Only download the illust width or height great then it")
}
