# PixivDownload

A tool to download pixiv illust.

## Usage

* Download all illustrations in user's bookmarks: `pixiv-dl download --cookie=abc --download-bookmarks-uids=1`
* Download illustrations of specified illustration id: `pixiv-dl download --download-illust-ids=1,2,3`
* Download all illustrations of specified users: `pixiv-dl download --download-artist-uids=1,2,3`
* Download all illustrations of specified users which bookmark count great then
  1000: `pixiv-dl download --download-artist-uids=1,2,3 --bookmark-gt=1000`

You can use `--service-mode` to run it as a service.

```
pixiv-dl -h
A tool to download pixiv illust

Usage:
  pixiv [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  download    Download illust from pixiv
  help        Help about any command
  info        Get the illust/user info

Flags:
      --config string       config file (default is ./pixiv.yaml and $HOME/pixiv.yaml)
      --cookie string       Your Cookies, only need the key-value 'PHPSESSID=abcxyz'
  -h, --help                help for pixiv
      --log-level string    Log level, choices: [DEBUG, INFO, WARNING, ERROR] (default "INFO")
      --log-path string     Log file path (default is stdout)
      --user-agent string   Http User-Agent header (default "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")

Use "pixiv [command] --help" for more information about a command.

```

download sub cmd:

```
pixiv-dl download -h
Download the illust from your bookmarks, all illust of your following,
all illust of the users id you specified, or from a illust id list.
You can run it as service mode by '--service-mode' flag, it will check
and download new illust periodically.

Usage:
  pixiv download [flags]

Flags:
      --bookmark-gt int                   Only download the illust bookmarks count great then it (default -1)
      --database-type string              Database to store the illust info, 'NONE' means not use database and not check illust exist, choices: ['NONE', 'SQLITE'] (default "SQLITE")
      --download-artist-uids strings      Download all illust of this user
      --download-bookmarks-uids strings   Download all bookmarks illust of this user
      --download-following-uids strings   Download all following user's illust of this user
      --download-illust-ids strings       Illust ids to download
      --download-parallel int32           Parallel number to download illust (default 10)
      --download-path string              Download file location (default "pixiv")
      --download-timeout-ms int32         Timeout for download illust (default 600000)
      --filename-pattern string           Filename pattern, all tag can use: ['user_id, 'user', 'id', 'title'] (default "{id}")
  -h, --help                              help for download
      --like-gt int                       Only download the illust like count great then it (default -1)
      --max-retries int32                 Max retry times (default 2147483647)
      --no-r18                            Not download R18 illust
      --only-p0                           Only download the first picture of the illust if a multi picture illust it
      --parse-parallel int32              Parallel number to get an parse illust info (default 5)
      --parse-timeout-ms int32            Timeout for get illust info (default 5000)
      --pixel-gt int                      Only download the illust width or height great then it (default -1)
      --retry-backoff-ms int32            Backoff time if request failed (default 10000)
      --scan-interval-sec int32           The interval to check new illust if run in service mode (default 3600)
      --service-mode                      Run as service mode, check and download new illust periodically
      --sqlite-path string                Sqlite file location if use sqlite database (default "storage")
      --user-block-list strings           Not download illust which user id in this list
      --user-white-list strings           Only download illust which user id in this list

Global Flags:
      --config string       config file (default is ./pixiv.yaml and $HOME/pixiv.yaml)
      --cookie string       Your Cookies, only need the key-value 'PHPSESSID=abcxyz'
      --log-level string    Log level, choices: [DEBUG, INFO, WARNING, ERROR] (default "INFO")
      --log-path string     Log file path (default is stdout)
      --user-agent string   Http User-Agent header (default "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")

```

If not specify a config file, it will find config file `pixiv.yaml` in `.` and `$HOME`.

### Config File

```yaml
log-path:
log-level: INFO

service-mode: false

database-type: sqlite
sqlite-path: storage
download-path: pixiv
filename-pattern: "{id}_{title}"
scan-interval-sec: 3600
parse-parallel: 5
download-parallel: 10
max-retries: 2147483647
retry-backoff-ms: 10000
parse-timeout-ms: 5000
download-timeout-ms: 600000

cookie: "PHPSESSID=ABCXYZ"
user-agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"

download-bookmarks-uids: [ 123456 ]
download-following-uids: [ ]
download-artist-uids: [ ]
download-illust-ids: [ ]

user-white-list: [ ]
user-block-list: [ ]

no-r18: false
only-p0: false
bookmark-gt: -1
like-gt: -1
pixel-gt: -1
```

* DatabaseType: pixiv-dl will store all illust info to the database, now only support 'sqlite'
* UserId: your user id, you can get it from your homepage URL.
* Cookie: your cookies, you can get it by F12
* FileNamePattern: default `{id}`
    * `{user_id}`: Illust user id
    * `{user}`: Illust user name
    * `{id}`: The "{id}" include pages, it looks like "123456_p0"
    * `{title}`: Illust title
* UserWhiteList/UserBlockList: filter illust in your bookmarks, block list will be ignored when white list is not
  empty.

All config item will be read from env, the env var name MUST be start with `PIXIV_` and in
upper case(env key split by `_`, e.g. `PIXIV_DOWNLOAD_PATH`).

### Docker

```shell
docker run -d \
  --name=pixiv-dl \
  -e PIXIV_DOWNLOAD_BOOKMARKS_UIDS=123456 \
  -e PIXIV_COOKIE=PHPSESSID=XYZ \
  -v /path/to/storage/folder:/storage \
  -v /path/to/download/folder:/pixiv \
  --restart unless-stopped \
  littleneko/pixiv-dl:latest
```
