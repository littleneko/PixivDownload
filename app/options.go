package app

type PixivDlOptions struct {
	DatabaseType    string `mapstructure:"database-type"`
	SqlitePath      string `mapstructure:"sqlite-path"`
	DownloadPath    string `mapstructure:"download-path"`
	FilenamePattern string `mapstructure:"filename-pattern"`

	ScanIntervalSec   int32 `mapstructure:"scan-interval-sec"`
	ParseParallel     int32 `mapstructure:"parse-parallel"`
	DownloadParallel  int32 `mapstructure:"download-parallel"`
	MaxRetries        int32 `mapstructure:"max-retries"`
	RetryBackoffMs    int32 `mapstructure:"retry-backoff-ms"`
	ParseTimeoutMs    int32 `mapstructure:"parse-timeout-ms"`
	DownloadTimeoutMs int32 `mapstructure:"download-timeout-ms"`

	UserWhiteList []string `mapstructure:"user-white-list"`
	UserBlockList []string `mapstructure:"user-block-list"`

	Cookie    string `mapstructure:"cookie"`
	UserAgent string `mapstructure:"user-agent"`

	DownloadScope     []string `mapstructure:"download-scope"`
	UserId            string   `mapstructure:"user-id"`
	DownloadUserIds   []string `mapstructure:"download-user-ids"`
	DownloadIllustIds []string `mapstructure:"download-illust-ids"`

	NoR18  bool `mapstructure:"no-r18"`
	OnlyP0 bool `mapstructure:"only-p0"`

	HttpProxy  string `mapstructure:"http-proxy"`
	HttpsProxy string `mapstructure:"https-proxy"`
}
