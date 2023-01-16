/*
Copyright © 2023 litao.little@gmail.com

*/

package cmd

import (
	log "github.com/sirupsen/logrus"
	"math"
	"os"
	"os/signal"
	"pixiv/app"
	"strings"
	"syscall"

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
		options := getOptions()
		log.Infof("Use options: %s", options.ToJson(true))
		illustMgr, err := app.GetIllustInfoManager(options)
		cobra.CheckErr(err)

		if len(options.DownloadBookmarksUserIds) > 0 {
			downloadBookmarks(options, illustMgr)
		}
		if len(options.DownloadIllustIds) > 0 {
			downloadIllusts(options, illustMgr)
		}
		if len(options.DownloadArtistUserIds) > 0 {
			downloadArtists(options, illustMgr)
		}

		if options.ServiceMode {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
		}
	},
}

var downloadIllustCmd = &cobra.Command{
	Use:   "illust [illust id list]",
	Short: "Download by illust id",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			return
		}

		app.InitLog(viper.GetString("log-path"), viper.GetString("log-level"))

		options := getOptions()
		options.DownloadIllustIds = processArgs(args)
		log.Infof("Use options: %s", options.ToJson(true))
		illustMgr, err := app.GetIllustInfoManager(options)
		cobra.CheckErr(err)
		downloadIllusts(options, illustMgr)

		if options.ServiceMode {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
		}
	},
}

var downloadArtistCmd = &cobra.Command{
	Use:   "artist [user id list]",
	Short: "Download all illust of the user",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			return
		}
		app.InitLog(viper.GetString("log-path"), viper.GetString("log-level"))
		options := getOptions()
		options.DownloadArtistUserIds = processArgs(args)
		log.Infof("Use options: %s", options.ToJson(true))
		illustMgr, err := app.GetIllustInfoManager(options)
		cobra.CheckErr(err)
		downloadArtists(options, illustMgr)

		if options.ServiceMode {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
		}
	},
}

var downloadBookmarkCmd = &cobra.Command{
	Use:   "bookmark [user id list]",
	Short: "Download all bookmark illust of the user",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			return
		}
		app.InitLog(viper.GetString("log-path"), viper.GetString("log-level"))
		options := getOptions()
		options.DownloadBookmarksUserIds = processArgs(args)
		log.Infof("Use options: %s", options.ToJson(true))
		illustMgr, err := app.GetIllustInfoManager(options)
		cobra.CheckErr(err)
		downloadBookmarks(options, illustMgr)

		if options.ServiceMode {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
		}
	},
}

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"

func init() {
	downloadCmd.PersistentFlags().Bool("service-mode", false, "Run as a service, check and download new illust periodically")
	downloadCmd.PersistentFlags().String("database-type", "SQLITE", "Database to store the illust info, 'NONE' means not use database and not check illust exist, choices: ['NONE', 'SQLITE']")
	downloadCmd.PersistentFlags().String("sqlite-path", "storage", "Sqlite file location if use sqlite database")
	downloadCmd.PersistentFlags().String("download-path", "pixiv", "Download file location")
	downloadCmd.PersistentFlags().String("filename-pattern", "{id}", "Filename pattern, all tag can use: ['user_id, 'user', 'id', 'title']")
	downloadCmd.PersistentFlags().Int32("scan-interval-sec", 3600, "The interval to check new illust if run in service mode")
	downloadCmd.PersistentFlags().Int32("parse-parallel", 5, "Parallel number to get an parse illust info")
	downloadCmd.PersistentFlags().Int32("download-parallel", 10, "Parallel number to download illust")
	downloadCmd.PersistentFlags().Int32("max-retries", math.MaxInt32, "Max retry times")
	downloadCmd.PersistentFlags().Int32("retry-backoff-ms", 30000, "Backoff time if request failed")
	downloadCmd.PersistentFlags().Int32("parse-timeout-ms", 5000, "Timeout for get illust info")
	downloadCmd.PersistentFlags().Int32("download-timeout-ms", 600000, "Timeout for download illust")

	downloadCmd.Flags().StringSlice("dl-bookmarks-uids", []string{}, "Download all bookmarks illust of this user")
	downloadCmd.Flags().StringSlice("dl-following-uids", []string{}, "Download all following user's illust of this user")
	downloadCmd.Flags().StringSlice("dl-artist-uids", []string{}, "Download all illust of this user")
	downloadCmd.Flags().StringSlice("dl-illust-ids", []string{}, "Download illust of this id")

	downloadCmd.PersistentFlags().StringSlice("user-white-list", []string{}, "Only download illust which user id in this list")
	downloadCmd.PersistentFlags().StringSlice("user-block-list", []string{}, "Not download illust which user id in this list")

	downloadCmd.PersistentFlags().Bool("no-r18", false, "Not download R18 illust")
	downloadCmd.PersistentFlags().Bool("only-p0", false, "Only download the first picture of the illust if it's a multi picture illust")
	downloadCmd.PersistentFlags().Int("bookmark-gt", -1, "Only download the illust bookmarks count great then it")
	downloadCmd.PersistentFlags().Int("like-gt", -1, "Only download the illust like count great then it")
	downloadCmd.PersistentFlags().Int("pixel-gt", -1, "Only download the illust width or height great then it")

	downloadCmd.AddCommand(downloadIllustCmd)
	downloadCmd.AddCommand(downloadArtistCmd)
	downloadCmd.AddCommand(downloadBookmarkCmd)
}

func processArgs(args []string) []string {
	var pIds []string
	for _, arg := range args {
		ids := strings.Split(arg, ",")
		for _, id := range ids {
			idt := strings.TrimSpace(id)
			if len(idt) > 0 {
				pIds = append(pIds, idt)
			}
		}
	}
	return pIds
}

func getOptions() *app.PixivDlOptions {
	var options app.PixivDlOptions
	err := viper.Unmarshal(&options)
	if err != nil {
		log.Fatalf("Failed to read config file, msg: %s", err)
	}
	return &options
}

func downloadIllusts(options *app.PixivDlOptions, illustMgr app.IllustInfoManager) {
	downloader := app.NewIllustDownloader(options, illustMgr)
	if options.ServiceMode {
		go downloader.Start()
	} else {
		downloader.Start()
		downloader.Close()
	}
}

func downloadArtists(options *app.PixivDlOptions, illustMgr app.IllustInfoManager) {
	downloader := app.NewArtistDownloader(options, illustMgr)
	if options.ServiceMode {
		go downloader.Start()
	} else {
		downloader.Start()
		downloader.Close()
	}
}

func downloadBookmarks(options *app.PixivDlOptions, illustMgr app.IllustInfoManager) {
	downloader := app.NewBookmarksDownloader(options, illustMgr)
	if options.ServiceMode {
		go downloader.Start()
	} else {
		downloader.Start()
		downloader.Close()
	}
}
