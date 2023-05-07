package app

import (
	"math/rand"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	pixiv "github.com/littleneko/pixiv-api-go"
	log "github.com/sirupsen/logrus"
)

func isJsonUnmarshalError(err error) bool {
	_, ok := err.(*pixiv.ErrorJsonUnmarshal)
	return ok
}

const BookmarksPageLimit = 48

type PixivWorker interface {
	Run()
}

type pixivWorker struct {
	options   *PixivDlOptions
	illustMgr IllustInfoManager
	client    *pixiv.PixivClient

	userWhiteListFilter mapset.Set[pixiv.PixivID]
	userBlockListFilter mapset.Set[pixiv.PixivID]

	consumeCnt uint64
	produceCnt uint64
}

func newPixivWorker(options *PixivDlOptions, manager IllustInfoManager, timeout int32) *pixivWorker {
	worker := &pixivWorker{
		options:             options,
		illustMgr:           manager,
		userWhiteListFilter: mapset.NewSet[pixiv.PixivID](),
		userBlockListFilter: mapset.NewSet[pixiv.PixivID](),
		consumeCnt:          0,
		produceCnt:          0,
	}

	if len(options.Proxy) > 0 {
		proxy, _ := url.Parse(options.Proxy)
		worker.client = pixiv.NewPixivClientWithProxy(proxy, timeout)
	} else {
		worker.client = pixiv.NewPixivClient(timeout)
	}
	if len(options.Cookie) > 0 {
		cookieKV := strings.Split(options.Cookie, "=")
		if len(cookieKV) == 2 {
			worker.client.AddCookie(cookieKV[0], cookieKV[1])
		} else {
			worker.client.SetCookiePHPSESSID(options.Cookie)
		}
	}
	if len(options.UserAgent) > 0 {
		worker.client.SetUserAgent(options.UserAgent)
	}

	for _, uid := range options.UserWhiteList {
		worker.userWhiteListFilter.Add(pixiv.PixivID(uid))
	}
	for _, uid := range options.UserBlockList {
		worker.userBlockListFilter.Add(pixiv.PixivID(uid))
	}
	return worker
}

func (w *pixivWorker) retry(workFunc func() bool) {
	var retryTime int32 = 0
	for {
		ok := workFunc()
		if ok {
			break
		}
		if retryTime >= w.options.MaxRetries {
			break
		}
		retryTime++
		r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
		backoff := w.options.RetryBackoffMs + r.Int31n(w.options.RetryBackoffMs/10)
		time.Sleep(time.Duration(backoff) * time.Millisecond)
	}
}

// filterByUser return true means this illust should be skipped
func (w *pixivWorker) filterByUser(illustInfo *pixiv.IllustDigest) bool {
	// invalid user id
	if len(illustInfo.UserId) == 0 {
		return false
	}

	if w.userWhiteListFilter.Cardinality() > 0 && !w.userWhiteListFilter.Contains(illustInfo.UserId) {
		log.Debugf("[PixivWorker] Skip illust by UserWhiteList, %s", illustInfo.DigestString())
		return true
	}

	if w.userBlockListFilter.Cardinality() > 0 && w.userBlockListFilter.Contains(illustInfo.UserId) {
		log.Infof("[PixivWorker] Skip illust by UserBlockList, %s", illustInfo.DigestString())
		return true
	}

	return false
}

func (w *pixivWorker) filterByIllustInfo(illust *pixiv.IllustInfo) bool {
	if w.options.NoR18 && illust.R18 {
		log.Infof("[PixivWorker] Skip R18 illust: %s", illust.DigestString())
		return true
	}
	if w.options.OnlyP0 && illust.PageIdx > 0 {
		log.Infof("[PixivWorker] Skip no p0 illust: %s", illust.DigestString())
		return true
	}

	if w.options.BookmarkGt > 0 && illust.BookmarkCount > 0 && illust.BookmarkCount < w.options.BookmarkGt {
		log.Infof("[PixivWorker] Skip illust by bookmark count: %s", illust.DigestString())
		return true
	}
	if w.options.LikeGt > 0 && illust.LikeCount > 0 && illust.LikeCount < w.options.LikeGt {
		log.Infof("[PixivWorker] Skip illust by like count: %s", illust.DigestString())
		return true
	}

	if w.options.PixelGt > 0 && illust.Width > 0 && illust.Height > 0 &&
		illust.Width < w.options.PixelGt && illust.Height < w.options.PixelGt {
		log.Infof("[PixivWorker] Skip illust by width or height: %s", illust.DigestString())
		return true
	}

	return false
}

func (w *pixivWorker) checkIllustExist(id pixiv.PixivID) (bool, error) {
	exist := false
	err := Retry(func() error {
		var err error
		exist, err = w.illustMgr.IsIllustExist(string(id))
		return err
	}, 3)
	return exist, err
}

func (w *pixivWorker) checkIllustPageExist(id pixiv.PixivID, page int) (bool, error) {
	exist := false
	err := Retry(func() error {
		var err error
		exist, err = w.illustMgr.IsIllustPageExist(string(id), page)
		return err
	}, 3)
	return exist, err
}

func (w *pixivWorker) saveIllustInfo(illust *pixiv.IllustInfo, hash, filename string) error {
	return Retry(func() error {
		return w.illustMgr.SaveIllust(illust, hash, filename)
	}, 3)
}

func (w *pixivWorker) markIllustNotFound(id pixiv.PixivID) error {
	illust := &pixiv.IllustInfo{
		Id:        id,
		PageIdx:   0,
		PageCount: 1,
		Title:     "NOT FOUND",
	}
	return Retry(func() error {
		return w.illustMgr.SaveIllust(illust, "", "")
	}, 3)
}

func (w *pixivWorker) GetConsumeCnt() uint64 {
	return atomic.LoadUint64(&w.consumeCnt)
}

func (w *pixivWorker) ResetConsumeCnt() {
	atomic.StoreUint64(&w.consumeCnt, 0)
}

func (w *pixivWorker) GetProduceCnt() uint64 {
	return atomic.LoadUint64(&w.produceCnt)
}

func (w *pixivWorker) ResetProduceCnt() {
	atomic.StoreUint64(&w.produceCnt, 0)
}

func (w *pixivWorker) ResetCnt() {
	w.ResetProduceCnt()
	w.ResetConsumeCnt()
}

// BookmarksWorker process the input user id and output basic illust info of bookmarks
type BookmarksWorker struct {
	*pixivWorker

	input  <-chan pixiv.PixivID // input user id
	output chan<- *pixiv.IllustDigest
}

func NewBookmarksWorker(options *PixivDlOptions, illustMgr IllustInfoManager,
	input <-chan pixiv.PixivID, output chan<- *pixiv.IllustDigest) *BookmarksWorker {
	worker := &BookmarksWorker{
		pixivWorker: newPixivWorker(options, illustMgr, options.ParseTimeoutMs),
		input:       input,
		output:      output,
	}

	return worker
}

func (w *BookmarksWorker) Run() {
	go func() {
		for uid := range w.input {
			w.processInput(uid)
			atomic.AddUint64(&w.consumeCnt, 1)
		}
	}()
}

func (w *BookmarksWorker) processInput(uid pixiv.PixivID) {
	bookmarkClient := NewBookmarksPageClient(w.client, string(uid), BookmarksPageLimit)
	for {
		if !bookmarkClient.HasMorePage() {
			log.Infof("[BookmarksWorker] End scan all bookmarks for uid '%s'", uid)
			break
		}
		w.retry(func() bool {
			bmInfos, err := bookmarkClient.GetNextPageBookmarks()
			if err == pixiv.ErrNotFound || isJsonUnmarshalError(err) {
				log.Warningf("[BookmarksWorker] Skip bookmarks page, offset: %d, msg: %s", bookmarkClient.CurOffset(), err)
				return true
			}
			if err != nil {
				log.Warningf("[BookmarksWorker] Failed to get bookmarks, offset: %d, retry, msg: %s", bookmarkClient.CurOffset(), err)
				return false
			}
			err = w.processOutput(bmInfos)
			if err != nil {
				log.Warningf("[BookmarksWorker] Failed to process bookmarks, offset: %d, retry, msg: %s", bookmarkClient.CurOffset(), err)
				return false
			}
			log.Infof("[BookmarksWorker] Success get bookmarks, offset: %d, total: %d", bookmarkClient.CurOffset(), bookmarkClient.Total())
			return true
		})
		bookmarkClient.MoveToNextPage()
	}
}

func (w *BookmarksWorker) processOutput(bmInfo *pixiv.BookmarksInfo) error {
	for idx := range bmInfo.Works {
		illust := bmInfo.Works[idx]
		if w.filterByUser(illust) {
			continue
		}

		exist, err := w.checkIllustExist(illust.Id)
		if err != nil {
			log.Errorf("[BookmarksWorker] Failed to check illust exist, illust info: %s, msg: %s", illust.DigestString(), err)
			return err
		}
		if exist {
			log.Debugf("[BookmarksWorker] Skip exist illust, illust info: %s", illust.DigestString())
			continue
		}

		log.Infof("[BookmarksWorker] Success get bookmark illust info: %s", illust.DigestString())
		w.output <- illust
		atomic.AddUint64(&w.produceCnt, 1)
	}
	return nil
}

// ArtistWorker process the input user id and output basic illust info of all illust of this user
type ArtistWorker struct {
	*pixivWorker

	input  <-chan pixiv.PixivID // input user id
	output chan<- *pixiv.IllustDigest
}

func NewArtistWorker(options *PixivDlOptions, illustMgr IllustInfoManager,
	input <-chan pixiv.PixivID, output chan<- *pixiv.IllustDigest) *ArtistWorker {
	worker := &ArtistWorker{
		pixivWorker: newPixivWorker(options, illustMgr, options.ParseTimeoutMs),
		input:       input,
		output:      output,
	}

	return worker
}

func (w *ArtistWorker) Run() {
	go func() {
		for uid := range w.input {
			w.processInput(uid)
			atomic.AddUint64(&w.consumeCnt, 1)
		}
	}()
}

func (w *ArtistWorker) processInput(uid pixiv.PixivID) {
	w.retry(func() bool {
		illustIds, err := w.client.GetUserIllusts(string(uid))
		if err == pixiv.ErrNotFound || isJsonUnmarshalError(err) {
			log.Warningf("[ArtistWorker] Skip user: %s, msg: %s", uid, err)
			return true
		}
		if err != nil {
			log.Warningf("[ArtistWorker] Failed to get artist user %s, retry, msg: %s", uid, err)
			return false
		}

		log.Infof("[ArtistWorker] Success get user all ilusts, count: %d, ids: %+v", len(illustIds), illustIds)
		err = w.processOutput(illustIds)
		if err != nil {
			log.Warningf("[ArtistWorker] Failed to process artist user %s, retry, msg: %s", uid, err)
			return false
		}
		return true
	})
}

func (w *ArtistWorker) processOutput(illustIds []pixiv.PixivID) error {
	for _, id := range illustIds {
		exist, err := w.checkIllustExist(id)
		if err != nil {
			log.Errorf("[ArtistWorker] Failed to check illust exist, id: %s, msg: %s", id, err)
			return err
		}
		if exist {
			log.Debugf("[ArtistWorker] Skip exist illust, id: %s", id)
			continue
		}

		var illust = &pixiv.IllustDigest{
			Id:        id,
			PageCount: 1,
		}
		w.output <- illust
		atomic.AddUint64(&w.produceCnt, 1)
	}
	return nil
}

// IllustInfoWorker process the input basic illust info and output full illust info
type IllustInfoWorker struct {
	*pixivWorker
	input  <-chan *pixiv.IllustDigest
	output chan<- *pixiv.IllustInfo
}

func NewIllustInfoWorker(options *PixivDlOptions, illustMgr IllustInfoManager,
	input <-chan *pixiv.IllustDigest, output chan<- *pixiv.IllustInfo) *IllustInfoWorker {
	worker := &IllustInfoWorker{
		pixivWorker: newPixivWorker(options, illustMgr, options.ParseTimeoutMs),
		input:       input,
		output:      output,
	}
	return worker
}

func (w *IllustInfoWorker) Run() {
	for i := int32(0); i < w.options.ParseParallel; i++ {
		go func() {
			for illust := range w.input {
				w.processInput(illust)
				atomic.AddUint64(&w.consumeCnt, 1)
			}
			log.Info("[IllustInfoWorker] exit")
		}()
	}
}

func (w *IllustInfoWorker) processInput(illust *pixiv.IllustDigest) {
	w.retry(func() bool {
		exist, err := w.checkIllustExist(illust.Id)
		if err != nil {
			log.Errorf("[IllustInfoWorker] Failed to check illust exist, illust info: %s, msg: %s", illust.DigestString(), err)
			return false
		}
		if exist {
			log.Debugf("[IllustInfoWorker] Skip exist illust, illust info: %s", illust.DigestString())
			return true
		}

		illusts, err := w.client.GetIllustInfo(illust.Id, w.options.OnlyP0)
		if err == pixiv.ErrNotFound || isJsonUnmarshalError(err) {
			log.Warningf("[IllustInfoWorker] Skip illust: %s, msg: %s", illust.DigestString(), err)
			if err == pixiv.ErrNotFound {
				_ = w.markIllustNotFound(illust.Id)
			}
			return true
		}
		if err != nil {
			log.Warningf("[IllustInfoWorker] Failed to get illust info: %s, msg: %s", illust.DigestString(), err)
			return false
		}
		log.Debugf("[IllustInfoWorker] Success get illust info: %s", illusts[0].DigestString())
		w.processOutput(illusts)

		return true
	})
}

func (w *IllustInfoWorker) processOutput(illusts []*pixiv.IllustInfo) {
	for idx := range illusts {
		fullIllust := illusts[idx]
		if w.filterByIllustInfo(fullIllust) {
			continue
		}
		w.output <- fullIllust
		atomic.AddUint64(&w.produceCnt, 1)
	}
}

// IllustDownloadWorker process the input full illust info and download the illust to disk
type IllustDownloadWorker struct {
	*pixivWorker
	input <-chan *pixiv.IllustInfo
}

func NewIllustDownloadWorker(options *PixivDlOptions, illustMgr IllustInfoManager, illustChan <-chan *pixiv.IllustInfo) *IllustDownloadWorker {
	worker := &IllustDownloadWorker{
		pixivWorker: newPixivWorker(options, illustMgr, options.DownloadTimeoutMs),
		input:       illustChan,
	}
	return worker
}

func (w *IllustDownloadWorker) Run() {
	for i := int32(0); i < w.options.DownloadParallel; i++ {
		go func() {
			for illust := range w.input {
				w.processInput(illust)
				atomic.AddUint64(&w.consumeCnt, 1)
			}
			log.Info("[IllustDownloadWorker] exit")
		}()
	}
}

func FormatFileName(illust *pixiv.IllustInfo, pattern string) string {
	filename := filepath.Base(illust.Urls.Original)
	if len(pattern) == 0 {
		return filename
	}
	fileExt := filepath.Ext(filename)
	pid := filename[:len(filename)-len(fileExt)]

	newName := pattern
	newName = strings.Replace(newName, "{id}", pid, -1)
	newName = strings.Replace(newName, "{title}", StandardizeFileName(illust.Title), -1)
	newName = strings.Replace(newName, "{user_id}", string(illust.UserId), -1)
	newName = strings.Replace(newName, "{user}", StandardizeFileName(illust.UserName), -1)
	newName += fileExt
	return newName
}

func (w *IllustDownloadWorker) processInput(illust *pixiv.IllustInfo) {
	if len(illust.Urls.Original) == 0 {
		log.Warningf("[IllustDownloadWorker] Skip empty url illust: %s", illust.DigestString())
		return
	}

	filename := FormatFileName(illust, w.options.FilenamePattern)
	fullFilename := filepath.Join(w.options.DownloadPath, filename)
	w.retry(func() bool {
		exist, err := w.checkIllustPageExist(illust.Id, illust.PageIdx)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to check illust exist, illust info: %s, msg: %s", illust.DigestString(), err)
			// ignore error and download
		} else if exist {
			log.Debugf("[IllustDownloadWorker] Skip exist illust, illust info: %s", illust.DigestString())
			return true
		}

		start := time.Now()
		size, hash, err := w.client.DownloadIllust(illust.Urls.Original, fullFilename)
		if err == pixiv.ErrNotFound || isJsonUnmarshalError(err) {
			return true
		}
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to download illust and retry, %s, url: %s, msg: %s", illust.DigestString(), illust.Urls.Original, err)
			return false
		}

		err = w.saveIllustInfo(illust, hash, filename)
		if err != nil {
			log.Errorf("[IllustDownloadWorker] Failed to save illust info and retry, %s, msg: %s", illust.DigestString(), err)
			return false
		}
		elapsed := time.Since(start)
		log.Infof("[IllustDownloadWorker] Success download illust: %s, cost: %s, size: %dKB, filename: %s, URL: %s", illust.DigestString(), elapsed, size/1024, filename, illust.Urls.Original)
		return true
	})
}
