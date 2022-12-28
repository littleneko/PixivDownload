package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	UserId                     string
	Cookie                     string
	UserAgent                  string
	ScanInterval               uint64
	RetryInterval              uint64
	IllustInfoFetchWorkerCount int
	IllustDownloadWorkerCount  int
}

func main() {
	logrus.SetReportCaller(true)

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.SetDefault("ScanInterval", 3600)
	viper.SetDefault("RetryInterval", 5)
	viper.SetDefault("IllustInfoFetchWorkerCount", 5)
	viper.SetDefault("IllustDownloadWorkerCount", 10)
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Errorf("Failed to read config file, msg: %s", err.Error())
		return
	}

	var conf Config
	err = viper.Unmarshal(&conf)
	if err != nil {
		logrus.Errorf("Failed to read config file, msg: %s", err.Error())
		return
	}
	logrus.Infof("Use config: %+v", conf)

	workChan := make(chan *BookmarkWorks, 100)
	illustChan := make(chan *Illust, 100)

	illustWorker := NewIllustInfoFetchWorker(&conf, workChan, illustChan)
	illustWorker.run()

	illustDownloader := NewIllustDownloadWorker(&conf, illustChan)
	illustDownloader.run()

	worker := NewBookmarkFetchWorker(&conf, workChan)
	worker.run()
}
