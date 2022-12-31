package pkg

import (
	"math"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	LogToFile       bool
	LogPath         string
	LogLevel        string
	DatabaseType    string
	SqlitePath      string
	DownloadPath    string
	FileNamePattern string

	UserId    string
	Cookie    string
	UserAgent string

	ScanIntervalSec     uint32
	RetryIntervalSec    uint32
	MaxRetryTimes       uint32
	ParserWorkerCount   int
	DownloadWorkerCount int
	ParseTimeoutMs      int32
	DownloadTimeoutMs   int32

	UserIdWhiteList []string
	UserIdBlockList []string

	Proxy string
}

func GetConfig(file string) *Config {
	if len(file) > 0 {
		viper.SetConfigFile(file)
	} else {
		viper.SetConfigName("pixiv")
		viper.SetConfigType("toml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("config")
	}

	viper.SetEnvPrefix("PIXIV")
	viper.AutomaticEnv()
	_ = viper.BindEnv("FileNamePattern")
	_ = viper.BindEnv("UserId")
	_ = viper.BindEnv("Cookie")
	_ = viper.BindEnv("UserIdWhiteList")
	_ = viper.BindEnv("UserIdBlockList")

	viper.SetDefault("LogToFile", false)
	viper.SetDefault("LogPath", "log")
	viper.SetDefault("LogLevel", "INFO")
	viper.SetDefault("DatabaseType", "sqlite")
	viper.SetDefault("SqlitePath", "storage")
	viper.SetDefault("DownloadPath", "pixiv")

	viper.SetDefault("UserAgent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")

	viper.SetDefault("ScanIntervalSec", 3600)
	viper.SetDefault("RetryIntervalSec", 60)
	viper.SetDefault("MaxRetryTimes", math.MaxUint32)
	viper.SetDefault("ParserWorkerCount", 5)
	viper.SetDefault("DownloadWorkerCount", 10)
	viper.SetDefault("ParseTimeoutMs", 5000)
	viper.SetDefault("DownloadTimeoutMs", 60000)

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Warningf("Config file not found, use default config")
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
