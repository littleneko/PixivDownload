# PixivDownload

A tool to download pixiv illust. (Now only support download illust in your bookmarks.)

## Usage

```
Usage of ./pixiv-dl:
  -config string
    	config file name
```

If not specify a config file, it will find config file `pixiv.toml` in `.` and `/config`.

### Config File

```toml
LogToFile = false
LogPath = "log"
LogLevel = "INFO"
DatabaseType = "sqlite"
SqlitePath = "storage"
DownloadPath = "pixiv"
FileNamePattern = "{user_id}_{user}/{id}_{title}"

UserId = "123456"
Cookie = "PHPSESSID=xxx"
UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"
ScanIntervalSec = 3600
RetryIntervalSec = 60
MaxRetryTimes = 4294967295
ParserWorkerCount = 5
DownloadWorkerCount = 20
ParseTimeoutMs = 5000
DownloadTimeoutMs = 60000
UserIdWhiteList = ["2131660"]
UserIdBlockList = []
```

* DatabaseType: pixiv-dl will store all illust info to it, now only support 'sqlite'
* UserId: your user id, you can get it from your homepage URL.
* Cookie: your cookies, you can get it by F12
* FileNamePattern: default `{id}`
  * `{user_id}`: Illust user id
  * `{user}`: Illust user name
  * `{id}`: The "{id}" include pages, it looks like "123456_p0"
  * `{title}`: Illust title
* UserIdWhiteList/UserIdBlockList: filter illust in your bookmarks, block list will be ignored when white list is not empty.

All config item will be read from env if not set in config file, the env var name MUST be start with `PIXIV_` and in upper case(config item NOT split
by `_`, e.g. `PIXIV_DOWNLOADPATH`).

### Docker

```shell
docker run -d \
  --name=pixiv-dl \
  -e PIXIV_USERID=123456 \
  -e PIXIV_COOKIE=PHPSESSID=XYZ \
  -v /path/to/storage/folder:/storage \
  -v /path/to/download/folder:/pixiv \
  --restart unless-stopped \
  littleneko/pixiv-dl:latest
```

## TODO

1. support download all illust of a user.
2. support download illust by id or url.
3. support check database and data file, rename data file
4. get illust tag and other info