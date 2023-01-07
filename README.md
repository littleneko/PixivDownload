# PixivDownload

A tool to download pixiv illust. (Now only support download illust in your bookmarks.)

## Usage

```
./pixiv-dl -h
A tool to download pixiv illust

Usage:
  pixiv [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  download    Download illust from pixiv
  help        Help about any command

Flags:
      --config string      config file (default is ./pixiv.yaml and $HOME/pixiv.yaml)
  -h, --help               help for pixiv
      --log-level string   Log level, choices: [DEBUG, INFO, WARNING, ERROR] (default "INFO")
      --log-path string    Log file path (default is stdout)

Use "pixiv [command] --help" for more information about a command.

```

download sub cmd:

```
./pixiv-dl download -h
Download the illust from your bookmarks, all illust of your following,
all illust of the users id you specified, or from a illust id list.
You can run it as service mode by --service flag, it will check and download new illust periodically.

Usage:
  pixiv download [flags]

Flags:
      --cookie string                 Your Cookies, only need the 'PHPSESSID=abcxyz'
      --database-type string          Database to store the illust info, 'NONE' means not use database and not check illust exist, choices: ['NONE', 'SQLITE'] (default "SQLITE")
      --download-illust-ids strings   Illust id to download
      --download-parallel int32       Parallel to download illust (default 10)
      --download-path string          Download file location (default "pixiv")
      --download-scope strings        What to download, choices: ['ALL', 'BOOKMARKS', 'FOLLOWING', 'USER', 'ILLUST'] (default [ALL])
      --download-timeout-ms int32     Timeout for download illust (default 10000)
      --download-user-ids strings     Download all illust of this user
      --filename-pattern string       Filename pattern, all tag: ['user_id, 'user', 'id', 'title'] (default "{id}")
  -h, --help                          help for download
      --max-retries int32             Max retry times (default 2147483647)
      --no-r18                        Do not download R18 illust
      --only-p0                       Only download the first picture of the illust if a multi picture illust
      --parse-parallel int32          Parallel to get an parse illust info (default 5)
      --parse-timeout-ms int32        Timeout for get illust info (default 5000)
      --retry-backoff-ms int32        Backoff time if request failed (default 1000)
      --scan-interval-sec int32       The interval to check new illust if run in service mode (default 3600)
      --sqlite-path string            Sqlite file location if use sqlite database (default "storage")
      --user-agent string             Http User-Agent header
      --user-block-list strings       Illust user id in this list will skip to download
      --user-id string                Download all bookmarks or following user's illust, if download-scope include bookmarks/following
      --user-white-list strings       Only illust user id in this list will be download

Global Flags:
      --config string      config file (default is ./pixiv.yaml and $HOME/pixiv.yaml)
      --log-level string   Log level, choices: [DEBUG, INFO, WARNING, ERROR] (default "INFO")
      --log-path string    Log file path (default is stdout)

```

If not specify a config file, it will find config file `pixiv.yaml` in `.` and `$HOME`.

### Config File

```yaml
log-path:
log-level: INFO

database-type: sqlite
sqlite-path: storage
download-path: pixiv
filename-pattern: "{user_id}_{user}/{id}_{title}"
scan-interval-sec: 3600
parse-parallel: 5
download-parallel: 10
max-retries: 2147483647
retry-backoff-ms: 1000
parse-timeout-ms: 5000
download-timeout-ms: 10000
user-white-list: [ ]
user-block-list: [ ]


cookie: "PHPSESSID=ABCXYZ"
user-agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"

download-scope: [ "ALL" ]
user-id: 123456
download-user-ids: [ ]
download-illust-ids: [ ]
no-r18: false
only-p0: false
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

All config item will be read from env if not set in config file, the env var name MUST be start with `PIXIV_` and in
upper case(env key split by `_`, e.g. `PIXIV_DOWNLOAD_PATH`).

### Docker

```shell
docker run -d \
  --name=pixiv-dl \
  -e PIXIV_USER_ID=123456 \
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