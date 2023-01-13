package pkg

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	BookmarksUrl   = "https://www.pixiv.net/ajax/user/%s/illusts/bookmarks"
	IllustUrl      = "https://www.pixiv.net/ajax/illust/%s"
	IllustPagesUrl = "https://www.pixiv.net/ajax/illust/%s/pages"
)

const (
	BookmarksReferUrl      = "https://www.pixiv.net/users/%s/bookmarks/artworks"
	IllustReferUrl         = "https://www.pixiv.net/artworks/%s"
	IllustDownloadReferUrl = "https://www.pixiv.net"
)

var IllegalFileNameChar = [...]string{"*", "\"", "<", ">", "?", "\\", "|", "/", ":"}

func StandardizeFileName(name string) string {
	newName := name
	for _, c := range IllegalFileNameChar {
		newName = strings.Replace(newName, c, "_", -1)
	}
	return newName
}

type Worker struct {
	conf   *Config
	client *http.Client
	db     PixivDB
}

func (w *Worker) retry(workFunc func() bool) {
	for {
		ok := workFunc()
		if ok {
			break
		}
		time.Sleep(time.Duration(w.conf.RetryInterval) * time.Second)
	}
}

func (w *Worker) request(url string, refer string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Referer", refer)
	req.Header.Add("Cookie", w.conf.Cookie)
	req.Header.Add("User-Agent", w.conf.UserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == 404 {
		log.Warningf("404: %s", url)
		return resp, nil
	}
	if resp.StatusCode != 200 {
		return resp, errors.New(resp.Status)
	}
	return resp, nil
}

type BookmarkFetchWorker struct {
	Worker
	workChan chan<- *BookmarkWorks
	offset   int64
	total    int64

	userWhiteListFilter map[IdWrapper]struct{}
	userBlockListFilter map[IdWrapper]struct{}
}

const BookmarksLimit = 48

func NewBookmarkFetchWorker(conf *Config, workChan chan<- *BookmarkWorks, db PixivDB) BookmarkFetchWorker {
	worker := BookmarkFetchWorker{Worker: Worker{conf: conf, client: &http.Client{
		Timeout: 5 * time.Second,
	}, db: db}, workChan: workChan, total: -1}
	worker.userWhiteListFilter = make(map[IdWrapper]struct{})
	worker.userBlockListFilter = make(map[IdWrapper]struct{})
	for _, uid := range conf.UserIdWhiteList {
		worker.userWhiteListFilter[IdWrapper(uid)] = struct{}{}
	}
	for _, uid := range conf.UserIdBlockList {
		worker.userBlockListFilter[IdWrapper(uid)] = struct{}{}
	}
	return worker
}

func (w *BookmarkFetchWorker) NextUrl() string {
	params := url.Values{}
	params.Set("tag", "")
	params.Set("offset", strconv.FormatInt(w.offset, 10))
	params.Set("limit", strconv.FormatInt(BookmarksLimit, 10))
	params.Set("rest", "show")

	bmUrl, _ := url.Parse(fmt.Sprintf(BookmarksUrl, w.conf.UserId))
	bmUrl.RawQuery = params.Encode()

	return bmUrl.String()
}

func (w *BookmarkFetchWorker) MoveToNextPage() {
	w.offset += BookmarksLimit
}

func (w *BookmarkFetchWorker) HasMorePage() bool {
	return w.total == -1 || w.offset < w.total
}

func (w *BookmarkFetchWorker) filter(work *BookmarkWorks) bool {
	if len(w.userWhiteListFilter) > 0 {
		_, ok := w.userWhiteListFilter[work.UserId]
		if !ok {
			log.Debugf("[BookmarkFetchWorker] Skip illust by UserIdWhiteList, id: %s. title: %s, uid: %s, uname: %s", work.Id, work.Title, work.UserId, work.UserName)
			return true
		}
	}
	if len(w.userBlockListFilter) > 0 {
		_, ok := w.userBlockListFilter[work.UserId]
		if ok {
			log.Infof("[BookmarkFetchWorker] Skip illust by UserIdBlockList, id: %s. title: %s, uid: %s, uname: %s", work.Id, work.Title, work.UserId, work.UserName)
			return true
		}
	}
	return false
}

func (w *BookmarkFetchWorker) run() {
	refer := fmt.Sprintf(BookmarksReferUrl, w.conf.UserId)
	workFunc := func() bool {
		log.Infof("[BookmarkFetchWorker] Start to get bookmarks, offset: %d, limit: %d, total: %d", w.offset, BookmarksLimit, w.total)
		bookmarkUrl := w.NextUrl()
		resp, err := w.request(bookmarkUrl, refer)
		if err != nil {
			log.Warningf("[BookmarkFetchWorker] Failed to get bookmarks, retry, url: %s, msg: %s", bookmarkUrl, err)
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warningf("[BookmarkFetchWorker] Failed to get bookmarks, retry, url: %s, msg: %s", bookmarkUrl, err)
			return false
		}
		var bResp Resp
		_ = json.Unmarshal(body, &bResp)
		if bResp.Error {
			log.Warningf("[BookmarkFetchWorker] Failed to get bookmarks, retry, msg: %s", bResp.Message)
			return false
		}

		var bmBody BookmarkBody
		err = json.Unmarshal(bResp.Body, &bmBody)
		if err != nil {
			log.Errorf("[BookmarkFetchWorker] Failed to unmarshal json, skip, err: %s, raw: %s", err, bResp.Body)
			return true
		}
		w.total = int64(bmBody.Total)

		for idx := range bmBody.Works {
			work := &bmBody.Works[idx]
			if w.filter(work) {
				continue
			}

			exist := false
			err = Retry(func() error {
				exist, err = w.db.CheckIllust(string(work.Id), work.PageCount)
				return err
			}, 3)
			if err != nil {
				log.Errorf("[BookmarkFetchWorker] Failed to check illust exist, retry, id: %s, msg: %s", work.Id, err)
				return false
			}
			if exist {
				log.Debugf("[BookmarkFetchWorker] Illust exist, id: %s, title: %s, uid: %s, uname: %s", work.Id, work.Title, work.UserId, work.UserName)
			} else {
				log.Infof("[BookmarkFetchWorker] Success get bookmark work, id: %s, title: %s, uid: %s, uname: %s", work.Id, work.Title, work.UserId, work.UserName)
				w.workChan <- work
			}
		}
		return true
	}

	go func() {
		for {
			if !w.HasMorePage() {
				log.Infof("[BookmarkFetchWorker] Success get all bookmarks, wait for next round")
				w.offset = 0
				w.total = -1
				time.Sleep(time.Duration(w.conf.ScanInterval) * time.Second)
			}
			w.retry(workFunc)
			w.MoveToNextPage()
		}
	}()
}

type IllustInfoFetchWorker struct {
	Worker
	workChan   <-chan *BookmarkWorks
	illustChan chan<- *Illust
}

func NewIllustInfoFetchWorker(conf *Config, workChan <-chan *BookmarkWorks, illustChan chan<- *Illust, db PixivDB) IllustInfoFetchWorker {
	return IllustInfoFetchWorker{Worker: Worker{conf: conf, client: &http.Client{
		Timeout: 5 * time.Second,
	}, db: db}, workChan: workChan, illustChan: illustChan}
}

func (w *IllustInfoFetchWorker) run() {
	workFunc := func() {
		for work := range w.workChan {
			w.fetchIllustInfo(work)
		}
		log.Info("[IllustInfoFetchWorker] exit")
	}

	for i := 0; i < w.conf.ParserWorkerCount; i++ {
		go workFunc()
	}
}

func (w *IllustInfoFetchWorker) fetchIllustInfo(work *BookmarkWorks) {
	illust := w.fetchIllustBasicInfo(work)
	if illust.PageCount == 1 {
		log.Infof("[IllustInfoFetchWorker] Success get illust, id: %s, title: %s, page: %d, uid: %s, uname: %s", illust.Id, illust.Title, illust.CurPage, illust.UserId, illust.UserName)
		w.illustChan <- illust
	} else {
		illusts := w.fetchIllustAllPages(illust)
		for idx := range illusts {
			illustP := illusts[idx]
			log.Infof("[IllustInfoFetchWorker] Success get illust pages, id: %s, title: %s, page: %d, uid: %s, uname: %s", illustP.Id, illustP.Title, illustP.CurPage, illustP.UserId, illustP.UserName)
			w.illustChan <- illustP
		}
	}
}

func (w *IllustInfoFetchWorker) fetchIllustBasicInfo(work *BookmarkWorks) *Illust {
	log.Infof("[IllustInfoFetchWorker] Start to get illust info, id: %s, title: %s", work.Id, work.Title)
	var illust Illust
	illustUrl := fmt.Sprintf(IllustUrl, work.Id)
	refer := fmt.Sprintf(IllustReferUrl, work.Id)
	w.retry(func() bool {
		resp, err := w.request(illustUrl, refer)
		if err != nil {
			log.Warningf("[IllustInfoFetchWorker] Failed to get illust info, retry, id: %s, url: %s, msg: %s", work.Id, illustUrl, err)
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warningf("[IllustInfoFetchWorker] Failed to get illust info, retry, id: %s, url: %s, msg: %s", work.Id, illustUrl, err)
			return false
		}
		var iResp Resp
		_ = json.Unmarshal(body, &iResp)
		if iResp.Error {
			log.Warningf("[IllustInfoFetchWorker] Failed to get illust info, retry, id: %s, url: %s, err: %s", work.Id, illustUrl, iResp.Message)
			return false
		}

		err = json.Unmarshal(iResp.Body, &illust)
		if err != nil {
			log.Errorf("[IllustInfoFetchWorker] Failed to unmarshal json, skip, id: %s, err: %s, raw: %s", work.Id, err, iResp.Body)
			return true
		}
		return true
	})
	return &illust
}

func (w *IllustInfoFetchWorker) fetchIllustAllPages(seed *Illust) []*Illust {
	var illusts []*Illust
	illustUrl := fmt.Sprintf(IllustPagesUrl, seed.Id)
	refer := fmt.Sprintf(IllustReferUrl, seed.Id)
	w.retry(func() bool {
		resp, err := w.request(illustUrl, refer)
		if err != nil {
			log.Warningf("[IllustInfoFetchWorker] Failed to get illust pages, retry, id: %s, url: %s, msg: %s", seed.Id, illustUrl, err)
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warningf("[IllustInfoFetchWorker] Failed to get illust pages, retry, id: %s, url: %s, msg: %s", seed.Id, illustUrl, err)
			return false
		}
		var iResp Resp
		_ = json.Unmarshal(body, &iResp)
		if iResp.Error {
			log.Warningf("[IllustInfoFetchWorker] Failed to get illust page info, retry, id: %s, url: %s, err: %s", seed.Id, illustUrl, iResp.Message)
			return false
		}

		type IllustPagesUnit struct {
			Urls Urls `json:"urls"`
		}
		var illustPageBody []IllustPagesUnit
		err = json.Unmarshal(iResp.Body, &illustPageBody)
		if err != nil {
			log.Warningf("[IllustInfoFetchWorker] Failed to unmarshal json, skip, id: %s, err: %s, raw: %s", seed.Id, err, iResp.Body)
			return true
		}

		for idx := range illustPageBody {
			illust := *seed
			illust.CurPage = idx
			illust.Urls = illustPageBody[idx].Urls
			illusts = append(illusts, &illust)
		}

		return true
	})
	return illusts
}

type IllustDownloadWorker struct {
	Worker
	illustChan <-chan *Illust
}

func NewIllustDownloadWorker(conf *Config, illustChan <-chan *Illust, db PixivDB) IllustDownloadWorker {
	return IllustDownloadWorker{Worker: Worker{conf: conf, client: &http.Client{
		Timeout: 60 * time.Second,
	}, db: db}, illustChan: illustChan}
}

func (w *IllustDownloadWorker) run() {
	workFunc := func() {
		for illust := range w.illustChan {
			w.downloadIllust(illust)
		}
		log.Info("[IllustDownloadWorker] exit")
	}

	for i := 0; i < w.conf.DownloadWorkerCount; i++ {
		go workFunc()
	}
}

func (w *IllustDownloadWorker) formatFileName(illust *Illust) string {
	fileName := filepath.Base(illust.Urls.Original)
	if len(w.conf.FileNamePattern) == 0 {
		return fileName
	}
	extName := filepath.Ext(fileName)
	pid := fileName[:len(fileName)-len(extName)]

	var newName = w.conf.FileNamePattern
	newName = strings.Replace(newName, "{id}", pid, -1)
	newName = strings.Replace(newName, "{title}", StandardizeFileName(illust.Title), -1)
	newName = strings.Replace(newName, "{user_id}", string(illust.UserId), -1)
	newName = strings.Replace(newName, "{user}", StandardizeFileName(illust.UserName), -1)
	newName += extName
	return newName
}

func (w *IllustDownloadWorker) downloadIllust(illust *Illust) {
	fileName := w.formatFileName(illust)
	fullFileName := filepath.Join(w.conf.DownloadPath, fileName)
	fullDirName := filepath.Dir(fullFileName)
	w.retry(func() bool {
		resp, err := w.request(illust.Urls.Original, IllustDownloadReferUrl)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to download illust, retry, id: %s, url: %s, msg: %s", illust.Id, illust.Urls.Original, err)
			return false
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to download illust, retry, id: %s, url: %s, msg: %s", illust.Id, illust.Urls.Original, err)
			return false
		}

		err = CheckAndMkdir(fullDirName)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to create dir, dir: %s, msg: %s", fullDirName, err)
			return false
		}
		err = os.WriteFile(fullFileName, data, 0644)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to write illust, retry, id: %s, url: %s, msg: %s", illust.Id, illust.Urls.Original, err)
			return false
		}
		err = Retry(func() error {
			return w.db.SaveIllust(illust, fmt.Sprintf("%x", sha1.Sum(data)), fileName)
		}, 3)
		if err != nil {
			log.Errorf("[IllustDownloadWorker] Failed to write db, retry, id: %s, msg: %s", illust.Id, err)
			return false
		}
		log.Infof("[IllustDownloadWorker] Success download illust, id: %s, url: %s", illust.Id, illust.Urls.Original)
		return true
	})
}

func Start(conf *Config, db PixivDB) {
	workChan := make(chan *BookmarkWorks, 100)
	illustChan := make(chan *Illust, 100)

	illustWorker := NewIllustInfoFetchWorker(conf, workChan, illustChan, db)
	illustWorker.run()

	illustDownloader := NewIllustDownloadWorker(conf, illustChan, db)
	illustDownloader.run()

	worker := NewBookmarkFetchWorker(conf, workChan, db)
	worker.run()
}
