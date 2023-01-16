# PixivDownload

A tool to download pixiv illustrations.

## Usage

Command line usage:

* Download illustrations of specified illustration id: `pixiv-dl download illust 103842593,100259729`
  or `pixiv-dl download --dl-illust-ids=103842593,100259729`
* Download all illustrations of specified users: `pixiv-dl download artist 2131660`
  or `pixiv-dl download --dl-artist-uids=2131660`
* Download all illustrations of specified users which bookmark count great then
  1000: `pixiv-dl download artist 2131660 --bookmark-gt=1000`
* Download all illustrations in user's bookmarks: `pixiv-dl download bookmark 2131660`
  or `pixiv-dl download --dl-bookmarks-uids=2131660`

If you get empty result or 'Bad Request' error, try to set a cookie by `--cookie` and `--user-agent`.

**There are some illustrations that can only be displayed after login, so it is highly recommended to use it with
setting a cookie.**

You can use `--service-mode` to run as a service, it will check new illustrations periodically and download it.

For more information on how to use it, please use `pixiv-dl -h`, `pixiv-dl download -h` parameter to get.

### Use Proxy

pixiv-dl will auto use the `https_proxy` proxy setting from environment variable, you can set the environment
by `export http_proxy=http://127.0.0.1:7890`, if you want to use another proxy, run pixiv-dl
with `--proxy=http://ip:port` or `--proxy=socks5://ip:port`.

### Config

All command flags can be set by a yaml config file, If not specify a config file, it will find config file `pixiv.yaml`
in `.` and `$HOME`.

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

dl-bookmarks-uids: [ 123456 ]
dl-following-uids: [ ]
dl-artist-uids: [ ]
dl-illust-ids: [ ]

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
* UserWhiteList/UserBlockList: filter illust in your bookmarks, block list will be ignored when white list is not empty.

### ENV

All config item can be read from environment variable, the environment variable key *MUST* be upper case start
with `PIXIV_` and split by `_` (e.g. `PIXIV_DOWNLOAD_PATH`).

The priority is: 'command line flag' > 'environment' > 'config file'.

### Docker

Docker run as service mode.

```shell
docker run -d \
  --name=pixiv-dl \
  -e PIXIV_DL_BOOKMARKS_UIDS=123456 \
  -e PIXIV_COOKIE=PHPSESSID=XYZ \
  -v /path/to/storage/folder:/storage \
  -v /path/to/download/folder:/pixiv \
  --restart unless-stopped \
  littleneko/pixiv-dl:latest
```
