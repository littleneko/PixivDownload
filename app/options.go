package app

import "encoding/json"

type PixivDlOptions struct {
	Cookie    string `mapstructure:"cookie"`
	UserAgent string `mapstructure:"user-agent"`
	Proxy     string `mapstructure:"proxy"`

	ServiceMode bool `mapstructure:"service-mode"`

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

	DownloadBookmarksUserIds []string `mapstructure:"dl-bookmarks-uids"`
	DownloadFollowingUserIds []string `mapstructure:"dl-following-uids"`
	DownloadArtistUserIds    []string `mapstructure:"dl-artist-uids"`
	DownloadIllustIds        []string `mapstructure:"dl-illust-ids"`

	UserWhiteList []string `mapstructure:"user-white-list"`
	UserBlockList []string `mapstructure:"user-block-list"`

	NoR18      bool `mapstructure:"no-r18"`
	OnlyP0     bool `mapstructure:"only-p0"`
	BookmarkGt int  `mapstructure:"bookmark-gt"`
	LikeGt     int  `mapstructure:"like-gt"`
	PixelGt    int  `mapstructure:"pixel-gt"`
}

func (p *PixivDlOptions) ToJson(indent bool) string {
	var j []byte
	if indent {
		j, _ = json.MarshalIndent(p, "", "  ")
	} else {
		j, _ = json.Marshal(p)
	}
	return string(j)
}
