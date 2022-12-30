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
	var retryTime uint32 = 0
	for {
		ok := workFunc()
		if ok {
			break
		}
		if retryTime >= w.conf.MaxRetryTimes {
			break
		}
		retryTime++
		time.Sleep(time.Duration(w.conf.RetryIntervalSec) * time.Second)
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

func (w *Worker) requestResp(url string, refer string) (*Resp, error) {
	resp, err := w.request(url, refer)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var jResp Resp
	_ = json.Unmarshal(body, &jResp)
	if jResp.Error {
		return nil, errors.New(jResp.Message)
	}

	return &jResp, nil
}

type BookmarkFetchWorker struct {
	Worker
	workChan chan<- *BookmarkWorks
	offset   int32
	total    int32

	userWhiteListFilter map[IdWrapper]struct{}
	userBlockListFilter map[IdWrapper]struct{}
}

const BookmarksLimit = 48

func NewBookmarkFetchWorker(conf *Config, workChan chan<- *BookmarkWorks, db PixivDB) BookmarkFetchWorker {
	worker := BookmarkFetchWorker{
		Worker: Worker{
			conf: conf,
			client: &http.Client{
				Timeout: time.Duration(conf.ParseTimeoutMs) * time.Millisecond,
			},
			db: db},
		workChan:            workChan,
		total:               -1,
		userWhiteListFilter: map[IdWrapper]struct{}{},
		userBlockListFilter: map[IdWrapper]struct{}{},
	}

	for _, uid := range conf.UserIdWhiteList {
		worker.userWhiteListFilter[IdWrapper(uid)] = struct{}{}
	}
	for _, uid := range conf.UserIdBlockList {
		worker.userBlockListFilter[IdWrapper(uid)] = struct{}{}
	}
	return worker
}

func (w *BookmarkFetchWorker) Run() {
	go w.ProcessBookmarks()
}

func (w *BookmarkFetchWorker) ProcessBookmarks() {
	for {
		if !w.hasMorePage() {
			log.Infof("[BookmarkFetchWorker] Success get all bookmarks, wait for next round")
			w.offset = 0
			w.total = -1
			time.Sleep(time.Duration(w.conf.ScanIntervalSec) * time.Second)
		}
		w.retry(func() bool {
			bmBody, err := w.GetBookmarks()
			if err != nil {
				return false
			}
			if bmBody.Total > 0 {
				w.total = bmBody.Total
			}
			err = w.writeToQueue(bmBody)
			if err != nil {
				return false
			}
			return true
		})
		w.moveToNextPage()
	}
}

func (w *BookmarkFetchWorker) nextUrl() string {
	params := url.Values{}
	params.Set("tag", "")
	params.Set("offset", strconv.FormatInt(int64(w.offset), 10))
	params.Set("limit", strconv.FormatInt(BookmarksLimit, 10))
	params.Set("rest", "show")

	bmUrl, _ := url.Parse(fmt.Sprintf(BookmarksUrl, w.conf.UserId))
	bmUrl.RawQuery = params.Encode()

	return bmUrl.String()
}

func (w *BookmarkFetchWorker) moveToNextPage() {
	w.offset += BookmarksLimit
}

func (w *BookmarkFetchWorker) hasMorePage() bool {
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

func (w *BookmarkFetchWorker) checkIllustExist(work *BookmarkWorks) (bool, error) {
	exist := false
	err := Retry(func() error {
		var err error
		exist, err = w.db.CheckIllust(string(work.Id), work.PageCount)
		return err
	}, 3)
	return exist, err
}

func (w *BookmarkFetchWorker) writeToQueue(bmBody *BookmarkBody) error {
	for idx := range bmBody.Works {
		work := &bmBody.Works[idx]
		if w.filter(work) {
			continue
		}

		exist, err := w.checkIllustExist(work)
		if err != nil {
			log.Errorf("[BookmarkFetchWorker] Failed to check illust exist, retry, id: %s, msg: %s", work.Id, err)
			return err
		}
		if exist {
			log.Debugf("[BookmarkFetchWorker] Illust exist, id: %s, title: %s, uid: %s, uname: %s", work.Id, work.Title, work.UserId, work.UserName)
		} else {
			log.Infof("[BookmarkFetchWorker] Success get bookmark work, id: %s, title: %s, uid: %s, uname: %s", work.Id, work.Title, work.UserId, work.UserName)
			w.workChan <- work
		}
	}
	return nil
}

func (w *BookmarkFetchWorker) GetBookmarks() (*BookmarkBody, error) {
	log.Infof("[BookmarkFetchWorker] Start to get bookmarks, offset: %d, limit: %d, total: %d", w.offset, BookmarksLimit, w.total)
	bUrl := w.nextUrl()
	refer := fmt.Sprintf(BookmarksReferUrl, w.conf.UserId)
	resp, err := w.requestResp(bUrl, refer)
	if err != nil {
		log.Warningf("[BookmarkFetchWorker] Failed to get bookmarks, retry, url: %s, msg: %s", bUrl, err)
		return nil, err
	}

	var bmBody BookmarkBody
	err = json.Unmarshal(resp.Body, &bmBody)
	if err != nil {
		log.Errorf("[BookmarkFetchWorker] Failed to unmarshal json, skip, err: %s, raw: %s", err, resp.Body)
		bmBody.Total = -1
		bmBody.Works = bmBody.Works[:0]
	}
	return &bmBody, nil
}

type IllustInfoFetchWorker struct {
	Worker
	workChan   <-chan *BookmarkWorks
	illustChan chan<- *Illust
}

func NewIllustInfoFetchWorker(conf *Config, workChan <-chan *BookmarkWorks, illustChan chan<- *Illust, db PixivDB) IllustInfoFetchWorker {
	return IllustInfoFetchWorker{
		Worker: Worker{
			conf: conf,
			client: &http.Client{
				Timeout: time.Duration(conf.ParseTimeoutMs) * time.Millisecond,
			},
			db: db},
		workChan:   workChan,
		illustChan: illustChan,
	}
}

func (w *IllustInfoFetchWorker) Run() {
	for i := 0; i < w.conf.ParserWorkerCount; i++ {
		go func() {
			for work := range w.workChan {
				w.processIllustInfo(work)
			}
			log.Info("[IllustInfoFetchWorker] exit")
		}()
	}
}

func (w *IllustInfoFetchWorker) processIllustInfo(work *BookmarkWorks) {
	w.retry(func() bool {
		illusts, err := w.FetchIllustInfo(work)
		if err != nil {
			return false
		}
		for idx := range illusts {
			illustP := illusts[idx]
			log.Infof("[IllustInfoFetchWorker] Success get illust pages, id: %s, title: %s, page: %d, uid: %s, uname: %s", illustP.Id, illustP.Title, illustP.CurPage, illustP.UserId, illustP.UserName)
			w.illustChan <- illustP
		}
		return true
	})
}

func (w *IllustInfoFetchWorker) FetchIllustInfo(work *BookmarkWorks) ([]*Illust, error) {
	illust, err := w.fetchIllustBasicInfo(work)
	if err != nil {
		return nil, err
	}
	if illust.PageCount == 1 {
		return []*Illust{illust}, nil
	} else {
		return w.fetchIllustAllPages(illust)
	}
}

func (w *IllustInfoFetchWorker) fetchIllustBasicInfo(work *BookmarkWorks) (*Illust, error) {
	log.Infof("[IllustInfoFetchWorker] Start to get illust info, id: %s, title: %s", work.Id, work.Title)
	illustUrl := fmt.Sprintf(IllustUrl, work.Id)
	refer := fmt.Sprintf(IllustReferUrl, work.Id)
	iResp, err := w.requestResp(illustUrl, refer)
	if err != nil {
		log.Warningf("[IllustInfoFetchWorker] Failed to get illust info, retry, id: %s, url: %s, msg: %s", work.Id, illustUrl, err)
		return nil, err
	}

	var illust Illust
	err = json.Unmarshal(iResp.Body, &illust)
	if err != nil {
		log.Errorf("[IllustInfoFetchWorker] Failed to unmarshal json, skip, id: %s, err: %s, raw: %s", work.Id, err, iResp.Body)
		illust.Id = "0"
	}
	return &illust, nil
}

func (w *IllustInfoFetchWorker) fetchIllustAllPages(seed *Illust) ([]*Illust, error) {
	illustUrl := fmt.Sprintf(IllustPagesUrl, seed.Id)
	refer := fmt.Sprintf(IllustReferUrl, seed.Id)
	iResp, err := w.requestResp(illustUrl, refer)
	if err != nil {
		log.Warningf("[IllustInfoFetchWorker] Failed to get illust pages, retry, id: %s, url: %s, msg: %s", seed.Id, illustUrl, err)
		return nil, err
	}

	type IllustPagesUnit struct {
		Urls Urls `json:"urls"`
	}
	var illustPageBody []IllustPagesUnit
	err = json.Unmarshal(iResp.Body, &illustPageBody)
	if err != nil {
		log.Warningf("[IllustInfoFetchWorker] Failed to unmarshal json, skip, id: %s, err: %s, raw: %s", seed.Id, err, iResp.Body)
		return nil, nil
	}

	var illusts []*Illust
	for idx := range illustPageBody {
		illust := *seed
		illust.CurPage = idx
		illust.Urls = illustPageBody[idx].Urls
		illusts = append(illusts, &illust)
	}
	return illusts, nil
}

type IllustDownloadWorker struct {
	Worker
	illustChan <-chan *Illust
}

func NewIllustDownloadWorker(conf *Config, illustChan <-chan *Illust, db PixivDB) IllustDownloadWorker {
	return IllustDownloadWorker{Worker: Worker{conf: conf, client: &http.Client{
		Timeout: time.Duration(conf.DownloadTimeoutMs) * time.Millisecond,
	}, db: db}, illustChan: illustChan}
}

func (w *IllustDownloadWorker) Run() {
	for i := 0; i < w.conf.DownloadWorkerCount; i++ {
		go func() {
			for illust := range w.illustChan {
				w.DownloadIllust(illust)
			}
			log.Info("[IllustDownloadWorker] exit")
		}()
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

func (w *IllustDownloadWorker) writeFile(fileName string, data []byte) error {
	dirName := filepath.Dir(fileName)
	err := CheckAndMkdir(dirName)
	if err != nil {
		return err
	}
	return os.WriteFile(fileName, data, 0644)
}

func (w *IllustDownloadWorker) writeDB(illust *Illust, data []byte, fileName string) error {
	return Retry(func() error {
		return w.db.SaveIllust(illust, fmt.Sprintf("%x", sha1.Sum(data)), fileName)
	}, 3)
}

func (w *IllustDownloadWorker) DownloadIllust(illust *Illust) {
	fileName := w.formatFileName(illust)
	fullFileName := filepath.Join(w.conf.DownloadPath, fileName)
	w.retry(func() bool {
		resp, err := w.request(illust.Urls.Original, IllustDownloadReferUrl)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to download illust, retry, id: %s, url: %s, msg: %s", illust.Id, illust.Urls.Original, err)
			return false
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to download illust, retry, id: %s, url: %s, msg: %s", illust.Id, illust.Urls.Original, err)
			return false
		}

		err = w.writeFile(fullFileName, data)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to write illust, retry, id: %s, url: %s, msg: %s", illust.Id, illust.Urls.Original, err)
			return false
		}

		err = w.writeDB(illust, data, fileName)
		if err != nil {
			log.Errorf("[IllustDownloadWorker] Failed to write DB, retry, id: %s, msg: %s", illust.Id, err)
			return false
		}
		log.Infof("[IllustDownloadWorker] Success download illust, id: %s, url: %s", illust.Id, illust.Urls.Original)
		return true
	})
}

func Start(conf *Config, db PixivDB) {
	workChan := make(chan *BookmarkWorks, 100)
	illustChan := make(chan *Illust, 100)

	bookmarksFetchWorker := NewBookmarkFetchWorker(conf, workChan, db)
	bookmarksFetchWorker.Run()

	illustFetchWorker := NewIllustInfoFetchWorker(conf, workChan, illustChan, db)
	illustFetchWorker.Run()

	illustDownloadWorker := NewIllustDownloadWorker(conf, illustChan, db)
	illustDownloadWorker.Run()

}
