package app

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const BookmarksPageLimit = 48

type Worker struct {
	options   *PixivDlOptions
	illustMgr IllustInfoManager
	client    *PixivClient
}

func (w *Worker) retry(workFunc func() bool) {
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
		time.Sleep(time.Duration(w.options.RetryBackoffMs) * time.Microsecond)
	}
}

type BookmarksWorker struct {
	*Worker

	workChan chan<- *BasicIllustInfo

	userWhiteListFilter map[PixivID]struct{}
	userBlockListFilter map[PixivID]struct{}
}

func NewPixivBookmarksWorker(options *PixivDlOptions, illustMgr IllustInfoManager, workChan chan<- *BasicIllustInfo) *BookmarksWorker {
	worker := &BookmarksWorker{
		Worker: &Worker{
			options:   options,
			illustMgr: illustMgr,
			client:    NewPixivClient(options.Cookie, options.UserAgent, options.ParseTimeoutMs),
		},
		workChan:            workChan,
		userWhiteListFilter: map[PixivID]struct{}{},
		userBlockListFilter: map[PixivID]struct{}{},
	}

	for _, uid := range options.UserWhiteList {
		worker.userWhiteListFilter[PixivID(uid)] = struct{}{}
	}
	for _, uid := range options.UserBlockList {
		worker.userBlockListFilter[PixivID(uid)] = struct{}{}
	}
	return worker
}

func (pbw *BookmarksWorker) Run() {
	go pbw.ProcessBookmarks()
}

func (pbw *BookmarksWorker) ProcessBookmarks() {
	for {
		for _, uid := range pbw.options.BookmarksUserIds {
			pbw.ProcessUserBookmarks(uid)
		}
		log.Infof("[BookmarkFetchWorker] End scan all bookmarks, wait for next round")
		time.Sleep(time.Duration(pbw.options.ScanIntervalSec) * time.Second)
	}
}

func (pbw *BookmarksWorker) ProcessUserBookmarks(uid string) {
	bookmarksFetch := NewBookmarksFetcher(pbw.client, uid, BookmarksPageLimit)
	for {
		if !bookmarksFetch.HasMorePage() {
			log.Infof("[BookmarkFetchWorker] End scan all bookmarks for uid '%s'", uid)
			break
		}
		pbw.retry(func() bool {
			bmBody, err := bookmarksFetch.GetNextPageBookmarks()
			if err == ErrNotFound || err == ErrFailedUnmarshal {
				log.Warningf("[BookmarkFetchWorker] Skip bookmarks page, offset: %d, msg: %s", bookmarksFetch.CurOffset(), err)
				return true
			}
			if err != nil {
				log.Warningf("[BookmarkFetchWorker] Failed to get bookmarks, offset: %d, retry, msg: %s", bookmarksFetch.CurOffset(), err)
				return false
			}
			err = pbw.writeToQueue(bmBody)
			if err != nil {
				log.Warningf("[BookmarkFetchWorker] Failed to process bookmarks, offset: %d, retry, msg: %s", bookmarksFetch.CurOffset(), err)
				return false
			}
			log.Infof("[BookmarkFetchWorker] Success get bookmarks, offset: %d", bookmarksFetch.CurOffset())
			return true
		})
		bookmarksFetch.MoveToNextPage()
	}
}

func (pbw *BookmarksWorker) filter(work *BasicIllustInfo) bool {
	if len(pbw.userWhiteListFilter) > 0 {
		_, ok := pbw.userWhiteListFilter[work.UserId]
		if !ok {
			log.Debugf("[BookmarkFetchWorker] Skip illust by UserWhiteList, %s", work.DigestString())
			return true
		}
	}
	if len(pbw.userBlockListFilter) > 0 {
		_, ok := pbw.userBlockListFilter[work.UserId]
		if ok {
			log.Infof("[BookmarkFetchWorker] Skip illust by UserBlockList, %s", work.DigestString())
			return true
		}
	}
	return false
}

func (pbw *BookmarksWorker) checkIllustExist(work *BasicIllustInfo) (bool, error) {
	exist := false
	err := Retry(func() error {
		var err error
		exist, err = pbw.illustMgr.CheckIllust(string(work.Id), work.PageCount)
		return err
	}, 3)
	return exist, err
}

func (pbw *BookmarksWorker) writeToQueue(bmBody *BookmarksInfo) error {
	for idx := range bmBody.Works {
		work := &bmBody.Works[idx]
		if pbw.filter(work) {
			continue
		}

		exist, err := pbw.checkIllustExist(work)
		if err != nil {
			log.Errorf("[BookmarkFetchWorker] Failed to check illust exist, illust info: %s, msg: %s", work.DigestString(), err)
			return err
		}
		if exist {
			log.Debugf("[BookmarkFetchWorker] Skip exist illust, illust info: %s", work.DigestString())
		} else {
			log.Infof("[BookmarkFetchWorker] Success get bookmark illust info: %s", work.DigestString())
			pbw.workChan <- work
		}
	}
	return nil
}

type IllustInfoFetchWorker struct {
	*Worker
	workChan   <-chan *BasicIllustInfo
	illustChan chan<- *IllustInfo
}

func NewIllustInfoFetchWorker(options *PixivDlOptions, illustMgr IllustInfoManager, workChan <-chan *BasicIllustInfo, illustChan chan<- *IllustInfo) *IllustInfoFetchWorker {
	worker := &IllustInfoFetchWorker{
		Worker: &Worker{
			options:   options,
			illustMgr: illustMgr,
			client:    NewPixivClient(options.Cookie, options.UserAgent, options.ParseTimeoutMs),
		},
		workChan:   workChan,
		illustChan: illustChan,
	}
	return worker
}

func (w *IllustInfoFetchWorker) Run() {
	for i := int32(0); i < w.options.ParseParallel; i++ {
		go func() {
			for work := range w.workChan {
				w.processIllustInfo(work)
			}
			log.Info("[IllustInfoFetchWorker] exit")
		}()
	}
}

func (w *IllustInfoFetchWorker) processIllustInfo(work *BasicIllustInfo) {
	w.retry(func() bool {
		illusts, err := w.client.GetIllustInfo(work.Id, w.options.OnlyP0)
		if err == ErrNotFound || err == ErrFailedUnmarshal {
			log.Warningf("[IllustInfoFetchWorker] Skip illust: %s, msg: %s", work.DigestString(), err)
			return true
		}
		if err != nil {
			log.Warningf("[IllustInfoFetchWorker] Failed to get illust info: %s , msg: %s", work.DigestString(), err)
			return false
		}
		log.Infof("[IllustInfoFetchWorker] Success get illust info: %s", illusts[0].DigestString())
		for idx := range illusts {
			illustP := illusts[idx]
			if w.options.NoR18 && illustP.R18 {
				log.Infof("[IllustInfoFetchWorker] Skip R18 illust: %s", illustP.DigestString())
				continue
			}
			w.illustChan <- illustP
		}
		return true
	})
}

type IllustDownloadWorker struct {
	*Worker
	illustChan <-chan *IllustInfo
}

func NewIllustDownloadWorker(options *PixivDlOptions, illustMgr IllustInfoManager, illustChan <-chan *IllustInfo) *IllustDownloadWorker {
	worker := &IllustDownloadWorker{
		Worker: &Worker{
			options:   options,
			illustMgr: illustMgr,
			client:    NewPixivClient(options.Cookie, options.UserAgent, options.DownloadTimeoutMs),
		},
		illustChan: illustChan,
	}
	return worker
}

func (w *IllustDownloadWorker) Run() {
	for i := int32(0); i < w.options.DownloadParallel; i++ {
		go func() {
			for illust := range w.illustChan {
				w.DownloadIllust(illust)
			}
			log.Info("[IllustDownloadWorker] exit")
		}()
	}
}

func FormatFileName(illust *IllustInfo, pattern string) string {
	fileName := filepath.Base(illust.Urls.Original)
	if len(pattern) == 0 {
		return fileName
	}
	extName := filepath.Ext(fileName)
	pid := fileName[:len(fileName)-len(extName)]

	var newName = pattern
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

func (w *IllustDownloadWorker) saveIllustInfo(illust *IllustInfo, data []byte, fileName string) error {
	return Retry(func() error {
		return w.illustMgr.SaveIllust(illust, fmt.Sprintf("%x", sha1.Sum(data)), fileName)
	}, 3)
}

func (w *IllustDownloadWorker) DownloadIllust(illust *IllustInfo) {
	fileName := FormatFileName(illust, w.options.FilenamePattern)
	fullFileName := filepath.Join(w.options.DownloadPath, fileName)
	w.retry(func() bool {
		start := time.Now()
		data, err := w.client.getIllustData(illust.Urls.Original)
		if err == ErrNotFound || err == ErrFailedUnmarshal {
			return true
		}
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to download illust and retry, %s, url: %s, msg: %s", illust.DigestString(), illust.Urls.Original, err)
			return false
		}

		err = w.writeFile(fullFileName, data)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to write illust and retry, %s, url: %s, msg: %s", illust.DigestString(), illust.Urls.Original, err)
			return false
		}

		err = w.saveIllustInfo(illust, data, fileName)
		if err != nil {
			log.Errorf("[IllustDownloadWorker] Failed to save illust info and retry, %s, msg: %s", illust.DigestString(), err)
			return false
		}
		elapsed := time.Since(start)
		log.Infof("[IllustDownloadWorker] Success download illust: %s, filename: %s, url: %s, elapsed: %s", illust.DigestString(), fullFileName, illust.Urls.Original, elapsed)
		return true
	})
}

func Start(options *PixivDlOptions, illustMgr IllustInfoManager) error {
	workChan := make(chan *BasicIllustInfo, 100)
	illustChan := make(chan *IllustInfo, 100)

	if len(options.DownloadIllustIds) > 0 {
		go func() {
			for _, pid := range options.DownloadIllustIds {
				workChan <- &BasicIllustInfo{
					Id: PixivID(pid),
				}
			}
		}()
	}

	if len(options.BookmarksUserIds) > 0 {
		bookmarksFetchWorker := NewPixivBookmarksWorker(options, illustMgr, workChan)
		bookmarksFetchWorker.Run()
	}

	illustFetchWorker := NewIllustInfoFetchWorker(options, illustMgr, workChan, illustChan)
	illustFetchWorker.Run()

	illustDownloadWorker := NewIllustDownloadWorker(options, illustMgr, illustChan)
	illustDownloadWorker.Run()

	return nil
}
