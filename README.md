# PixivDownload

A tool download pixiv illust. (Now only support download "like")

## How to Use

```
Usage of ./pixiv-dl:
  -config string
    	config file name
```

If not specify a config file, it will find config file "pixiv.toml" in "." and "/config".

### Config

```toml
LogToFile = false
LogPath = "log"
LogLevel = "INFO"
DatabaseType = "sqlite"
SqlitePath = "database"
DownloadPath = "illust"
FileNamePattern = "{user_id}_{user}/{id}_{title}"

UserId = "123456"
Cookie = "PHPSESSID=xxx"
UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"
ScanInterval = 3600
RetryInterval = 60
ParserWorkerCount = 5
DownloadWorkerCount = 10
UserIdWhiteList = ["2131660"]
UserIdBlockList = []
```

* FileNamePattern: all tag can use, "{user_id}", "{user}", "{id}", "{title}" (The "{id}" include pages, it looks like "
  123456_p0"), default "{id}"

All config will be read from env if not set, the env var name will be start with "PIXIV_" and upper case(NOT split
by '_').

### Docker

```shell
docker run \
 --env=PIXIV_USERID=123456 \
 --env=PIXIV_COOKIE=PHPSESSID=XXX \
 --volume=pixiv/database:/database
 --volume=pixiv/illust:/illust
 pixiv-dl:latest
```

## TODO

1. support download all illust of a user.