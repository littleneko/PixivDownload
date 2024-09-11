# PixivDownload

A tool to download pixiv illustrations.

[![Build Status](https://github.com/littleneko/pixivdownload/actions/workflows/release.yml/badge.svg)](https://github.com/littleneko/pixivdownload/actions)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/littleneko/pixivdownload.svg)](https://github.com/littleneko/pixivdownload/tags)
[![GitHub release](https://img.shields.io/github/release/littleneko/pixivdownload.svg)](https://github.com/littleneko/pixivdownload/releases)
[![GitHub license](https://img.shields.io/github/license/littleneko/pixivdownload.svg)](https://github.com/littleneko/pixivdownload/blob/main/LICENSE)

## Usage

命令行使用:

* 下载指定 id 的插画: `pixiv-dl download illust 103842593,100259729`
  或是 `pixiv-dl download --dl-illust-ids=103842593,100259729`
* 下载某个用户所有的插画: `pixiv-dl download artist 2131660` 或是 `pixiv-dl download --dl-artist-uids=2131660`
* 下载某个用户所有收藏数量大于 1000 的插画: `pixiv-dl download artist 2131660 --bookmark-gt=1000`
* 下载某个用户收藏的插画: `pixiv-dl download bookmark 2131660` 或是 `pixiv-dl download --dl-bookmarks-uids=2131660`

如果返回了空结果或是 Bad Request 错误, 请尝试使用 cookies 登陆: 使用参数 `--cookie` 和 `--user-agent`.

**因为有些插画必须登陆才能看到，所以强烈建议使用 cookies 登陆后使用.**

上面所列出的所有命令将在执行完下载任务后退出, 想要一直定时检查是否有新插画并下载, 请使用 `--service-mode`
参数并使用 `--scan-interval-sec` 设置定时扫描时间间隔.

更多使用使用方法详见 `pixiv-dl -h` 和 `pixiv-dl download -h`.

### 使用代理

pixiv-dl 会从环境变量中读取 `https_proxy` 代理信息, 如果想要使用自定义的 proxy, 可以使用 `--proxy=http://ip:port`
或是 `--proxy=socks5://ip:port` 配置代理信息.

### 配置文件

所有命令行参数都会从 yaml 配置文件中读取, 如果在启动时没有使用 `--config` 指定配置文件, 将从当前目录和 HOME
目录寻找 `pixiv.yaml` 文件作为配置文件.

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

* database-type: 存储插画元数据和判断是否已经下载过的数据库, 默认使用 sqlite, 目前只支持 sqlite， 如果配置为 'NONE',
  将不使用任何数据库也不判断是否重复
* filename-pattern: default `{id}`
    * `{user_id}`: 插画作者 id
    * `{user}`: 插画作者 name
    * `{id}`: 插画 id, 包括 page_idx, 类似 '123456_p0'
    * `{title}`: 插画名称, 对于一些特殊字符和空格都会替换成 '_'
* dl-bookmarks-uids: 下载指定用户的"收藏", 支持多个
* dl-artist-uids: 下载指定用户所有的插画, 支持多个
* dl-illust-ids: 下载指定 id 的插画, 支持多个
* dl-following-uids: 暂不支持

  > 上面 4 个参数可以同时提供

### 环境变量配置

所有配置项都会从环境变量中读取, 环境变量以 `PIXIV_` 开头, 并且使用 `_` 分割 (e.g. `PIXIV_DOWNLOAD_PATH`).

配置优先级: 命令行参数 > 环境变量 > 配置文件

### Docker

Docker run as service mode. 
Note that pixiv-dl runs as UID 1000 and GID 1000 by default. These may be altered with the `PUID` and `PGID` environment variables.

```shell
docker run -d \
  --name=pixiv-dl \
  -e PUID=1000 \
  -e PGID=1000 \
  -e PIXIV_DL_BOOKMARKS_UIDS=123456 \
  -e PIXIV_COOKIE=PHPSESSID=XYZ \
  -v /path/to/storage/folder:/storage \
  -v /path/to/download/folder:/pixiv \
  --restart unless-stopped \
  littleneko/pixiv-dl:latest
```
