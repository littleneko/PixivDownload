package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

type Resp struct {
	Error   bool            `json:"error"`
	Message string          `json:"message"`
	Body    json.RawMessage `json:"body"`
}

type BookmarkWorks struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Url       string `json:"url"`
	UserId    string `json:"userId"`
	UserName  string `json:"userName"`
	PageCount int    `json:"pageCount"`
}

type BookmarkBody struct {
	Total int             `json:"total"`
	Works []BookmarkWorks `json:"works"`
}

type Urls struct {
	Mini     string `json:"mini"`
	Thumb    string `json:"thumb"`
	Small    string `json:"small"`
	Regular  string `json:"regular"`
	Original string `json:"original"`
}

type Illust struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Urls        Urls   `json:"urls"`
	//Tags        json.RawMessage `json:"tags"`
	UserId      string `json:"userId"`
	UserName    string `json:"userName"`
	UserAccount string `json:"userAccount"`
	CreatedDate string `json:"createdDate"`
	PageCount   int    `json:"pageCount"`
	CurPage     int    `json:"curPage"`
}

const (
	BookmarksUrl   = "https://www.pixiv.net/ajax/user/%s/illusts/bookmarks"
	IllustUrl      = "https://www.pixiv.net/ajax/illust/%s"
	IllustPagesUrl = "https://www.pixiv.net/ajax/illust/%s/pages"
)

const (
	BookmarksRefer = "https://www.pixiv.net/users/%s/bookmarks/artworks"
	IllustRefer    = "https://www.pixiv.net/artworks/%s"
)

type Worker struct {
	conf   *Config
	client *http.Client
}

func (w *Worker) newGetRequest(url string, refer string) *http.Request {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Referer", refer)
	req.Header.Add("Cookie", w.conf.Cookie)
	req.Header.Add("User-Agent", w.conf.UserAgent)

	return req
}

func (w *Worker) retry(workFunc func() bool) {
	for {
		ret := workFunc()
		if !ret {
			break
		}
		time.Sleep(time.Duration(w.conf.RetryInterval) * time.Second)
	}
}

func (w *Worker) Request(url string, refer string, workType string) (*http.Response, bool /* need retry*/) {
	logrus.Infof("Start %s from: %s", workType, url)
	req := w.newGetRequest(url, refer)
	resp, err := w.client.Do(req)
	if err != nil {
		logrus.Warningf("Failed to request %s, retry, err: %s", workType, err.Error())
		return nil, true
	}
	if resp.StatusCode == 404 {
		logrus.Warningf("Not Found %s, skip, status: %s", workType, resp.Status)
		return nil, false
	}
	if resp.StatusCode != 200 {
		logrus.Warningf("Failed to request %s, retry, status: %s", workType, resp.Status)
		return nil, true
	}
	return resp, false
}

type IllustInfoFetchWorker struct {
	Worker
	workChan   <-chan *BookmarkWorks
	illustChan chan<- *Illust
}

func NewIllustInfoFetchWorker(conf *Config, workChan <-chan *BookmarkWorks, illustChan chan<- *Illust) IllustInfoFetchWorker {
	return IllustInfoFetchWorker{Worker: Worker{conf: conf, client: &http.Client{}}, workChan: workChan, illustChan: illustChan}
}

func (w *IllustInfoFetchWorker) run() {
	workFunc := func() {
		for work := range w.workChan {
			w.fetchIllustInfo(work)
		}
		logrus.Info("IllustInfoFetchWorker exit")
	}

	for i := 0; i < w.conf.IllustInfoFetchWorkerCount; i++ {
		go workFunc()
	}
}

func (w *IllustInfoFetchWorker) fetchIllustInfo(work *BookmarkWorks) {
	illust := w.fetchIllustBasicInfo(work)
	if illust.PageCount == 1 {
		logrus.Infof("Success get illust, %+v", *illust)
		w.illustChan <- illust
	} else {
		illusts := w.fetchIllustAllPages(illust)
		for idx := range illusts {
			illustP := illusts[idx]
			logrus.Infof("Success get illust pages, %+v", *illustP)
			w.illustChan <- illustP
		}
	}
}

func (w *IllustInfoFetchWorker) fetchIllustBasicInfo(work *BookmarkWorks) *Illust {
	logrus.Infof("Start to get illust info, id: %s, title: %s", work.Id, work.Title)
	var illust Illust
	w.retry(func() bool {
		resp, retry := w.Request(fmt.Sprintf(IllustUrl, work.Id), fmt.Sprintf(IllustRefer, work.Id), "get illust info")
		if resp == nil {
			return retry
		}

		body, err := io.ReadAll(resp.Body)
		var iResp Resp
		_ = json.Unmarshal(body, &iResp)
		if iResp.Error {
			logrus.Warningf("Failed to get bookmarks, retry, err: %s", iResp.Message)
			return true
		}

		err = json.Unmarshal(iResp.Body, &illust)
		if err != nil {
			logrus.Warningf("Failed to unmarshal json, skip, err: %s", err.Error())
			return false
		}
		return false
	})
	return &illust
}

func (w *IllustInfoFetchWorker) fetchIllustAllPages(seed *Illust) []*Illust {
	var illusts []*Illust
	w.retry(func() bool {
		resp, retry := w.Request(fmt.Sprintf(IllustPagesUrl, seed.Id), fmt.Sprintf(IllustRefer, seed.Id), "get illust all page")
		if resp == nil {
			return retry
		}

		body, err := io.ReadAll(resp.Body)
		var iResp Resp
		_ = json.Unmarshal(body, &iResp)
		if iResp.Error {
			logrus.Warningf("Failed to get illust page info, retry, err: %s", iResp.Message)
			return true
		}

		type IllustPagesUnit struct {
			Urls Urls `json:"urls"`
		}
		var illustPageBody []IllustPagesUnit
		err = json.Unmarshal(iResp.Body, &illustPageBody)
		if err != nil {
			logrus.Warningf("Failed to unmarshal json, skip, err: %s", err.Error())
			return false
		}

		for idx := range illustPageBody {
			illust := *seed
			illust.CurPage = idx
			illust.Urls = illustPageBody[idx].Urls
			illusts = append(illusts, &illust)
		}

		return false
	})
	return illusts
}

type BookmarkFetchWorker struct {
	Worker
	workChan chan<- *BookmarkWorks
	offset   int64
	total    int64
}

const BookmarksLimit = 48

func NewBookmarkFetchWorker(conf *Config, workChan chan<- *BookmarkWorks) BookmarkFetchWorker {
	return BookmarkFetchWorker{Worker: Worker{conf: conf, client: &http.Client{}}, workChan: workChan, total: -1}
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

func (w *BookmarkFetchWorker) run() {
	workFunc := func() bool {
		resp, retry := w.Request(w.NextUrl(), fmt.Sprintf(BookmarksRefer, w.conf.UserId), "get bookmarks")
		if resp == nil {
			return retry
		}

		body, err := io.ReadAll(resp.Body)
		var bResp Resp
		_ = json.Unmarshal(body, &bResp)
		if bResp.Error {
			logrus.Warningf("Failed to get bookmarks, retry, err: %s", bResp.Message)
			return true
		}

		var bmBody BookmarkBody
		err = json.Unmarshal(bResp.Body, &bmBody)
		if err != nil {
			logrus.Warningf("Failed to unmarshal json, skip, err: %s", err.Error())
			w.MoveToNextPage()
			return false
		}
		w.total = int64(bmBody.Total)

		for idx := range bmBody.Works {
			work := &bmBody.Works[idx]
			logrus.Infof("Success get bookmarks info, id: %s, title: %s", work.Id, work.Title)
			w.workChan <- work
		}
		w.MoveToNextPage()
		return false
	}

	for {
		if !w.HasMorePage() {
			logrus.Infof("Success get all bookmarks")
			time.Sleep(time.Duration(w.conf.ScanInterval) * time.Second)
		}
		w.retry(workFunc)
	}
}

type IllustDownloadWorker struct {
	Worker
	illustChan <-chan *Illust
}

func NewIllustDownloadWorker(conf *Config, illustChan <-chan *Illust) IllustDownloadWorker {
	return IllustDownloadWorker{Worker: Worker{conf: conf, client: &http.Client{}}, illustChan: illustChan}
}

func (w *IllustDownloadWorker) run() {
	workFunc := func() {
		for illust := range w.illustChan {
			w.downloadIllust(illust)
		}
		logrus.Info("IllustDownloadWorker exit")
	}

	for i := 0; i < w.conf.IllustDownloadWorkerCount; i++ {
		go workFunc()
	}
}

func (w *IllustDownloadWorker) downloadIllust(illust *Illust) {
	w.retry(func() bool {
		resp, retry := w.Request(illust.Urls.Original, "https://www.pixiv.net/", "download illust")
		if resp == nil {
			return retry
		}
		//io.ReadAll(resp.Body)

		return false
	})
}
