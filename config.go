package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	LogPath             string
	LogLevel            string
	DatabaseType        string
	SqlitePath          string
	UserId              string
	Cookie              string
	UserAgent           string
	ScanInterval        uint64
	RetryInterval       uint64
	ParserWorkerCount   int
	DownloadWorkerCount int
	DownloadPath        string
	FileNamePattern     string
	Proxy               string
	UserIdWhiteList     []string
	UserIdBlockList     []string
}

func GetConfig(file string) *Config {
	viper.SetConfigFile(file)

	viper.SetDefault("LogPath", "log")
	viper.SetDefault("LogLevel", "INFO")
	viper.SetDefault("DatabaseType", "sqlite")
	viper.SetDefault("SqlitePath", "database")
	viper.SetDefault("ScanInterval", 3600)
	viper.SetDefault("RetryInterval", 60)
	viper.SetDefault("ParserWorkerCount", 5)
	viper.SetDefault("DownloadWorkerCount", 10)
	viper.SetDefault("DownloadPath", "pixiv")
	viper.SetDefault("UserAgent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Warningf("config file not found, use default config")
		} else {
			// Config file was found but another error was produced
			log.Fatalf("Failed to read config file, msg: %s", err)
		}
	}

	var conf Config
	err = viper.Unmarshal(&conf)
	if err != nil {
		log.Fatalf("Failed to read config file, msg: %s", err)
	}
	log.Infof("Use config: %+v", conf)
	return &conf
}
