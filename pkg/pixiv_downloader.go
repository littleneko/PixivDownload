package pkg

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
	conf      *Config
	illustMgr IllustInfoManager
	client    *PixivClient
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

type PixivBookmarksWorker struct {
	Worker

	workChan chan<- *BookmarkWork
	offset   int32
	total    int32

	userWhiteListFilter map[PixivIDType]struct{}
	userBlockListFilter map[PixivIDType]struct{}
}

func NewPixivBookmarksWorker(conf *Config, illustMgr IllustInfoManager, workChan chan<- *BookmarkWork) *PixivBookmarksWorker {
	worker := PixivBookmarksWorker{
		Worker: Worker{
			conf:      conf,
			illustMgr: illustMgr,
			client:    NewPixivClient(conf.Cookie, conf.UserAgent, conf.ParseTimeoutMs),
		},
		workChan:            workChan,
		offset:              0,
		total:               -1,
		userWhiteListFilter: map[PixivIDType]struct{}{},
		userBlockListFilter: map[PixivIDType]struct{}{},
	}

	for _, uid := range conf.UserIdWhiteList {
		worker.userWhiteListFilter[PixivIDType(uid)] = struct{}{}
	}
	for _, uid := range conf.UserIdBlockList {
		worker.userBlockListFilter[PixivIDType(uid)] = struct{}{}
	}
	return &worker
}

func (pbw *PixivBookmarksWorker) Run() {
	go pbw.ProcessBookmarks()
}

func (pbw *PixivBookmarksWorker) ProcessBookmarks() {
	for {
		if !pbw.hasMorePage() {
			log.Infof("[BookmarkFetchWorker] End scan all bookmarks, wait for next round")
			pbw.offset = 0
			pbw.total = -1
			time.Sleep(time.Duration(pbw.conf.ScanIntervalSec) * time.Second)
		}
		pbw.retry(func() bool {
			bmBody, err := pbw.client.GetBookmarks(pbw.conf.UserId, pbw.offset, BookmarksPageLimit)
			if err == ErrNotFound || err == ErrFailedUnmarshal {
				log.Warningf("[BookmarkFetchWorker] Skip bookmarks page, offset: %d, msg: %s", pbw.offset, err)
				return true
			}
			if err != nil {
				log.Warningf("[BookmarkFetchWorker] Failed to get bookmarks, offset: %d, retry, msg: %s", pbw.offset, err)
				return false
			}
			if bmBody.Total > 0 {
				pbw.total = bmBody.Total
			}
			err = pbw.writeToQueue(bmBody)
			if err != nil {
				log.Warningf("[BookmarkFetchWorker] Failed to process bookmarks, offset: %d, retry, msg: %s", pbw.offset, err)
				return false
			}
			return true
		})
		pbw.moveToNextPage()
	}
}

func (pbw *PixivBookmarksWorker) moveToNextPage() {
	pbw.offset += BookmarksPageLimit
}

func (pbw *PixivBookmarksWorker) hasMorePage() bool {
	return pbw.total == -1 || pbw.offset < pbw.total
}

func (pbw *PixivBookmarksWorker) filter(work *BookmarkWork) bool {
	if len(pbw.userWhiteListFilter) > 0 {
		_, ok := pbw.userWhiteListFilter[work.UserId]
		if !ok {
			log.Debugf("[BookmarkFetchWorker] Skip illust by UserIdWhiteList, %+v", *work)
			return true
		}
	}
	if len(pbw.userBlockListFilter) > 0 {
		_, ok := pbw.userBlockListFilter[work.UserId]
		if ok {
			log.Infof("[BookmarkFetchWorker] Skip illust by UserIdBlockList, %+v", *work)
			return true
		}
	}
	return false
}

func (pbw *PixivBookmarksWorker) checkIllustExist(work *BookmarkWork) (bool, error) {
	exist := false
	err := Retry(func() error {
		var err error
		exist, err = pbw.illustMgr.CheckIllust(string(work.Id), work.PageCount)
		return err
	}, 3)
	return exist, err
}

func (pbw *PixivBookmarksWorker) writeToQueue(bmBody *BookmarksBody) error {
	for idx := range bmBody.Works {
		work := &bmBody.Works[idx]
		if pbw.filter(work) {
			continue
		}

		exist, err := pbw.checkIllustExist(work)
		if err != nil {
			log.Errorf("[BookmarkFetchWorker] Failed to check illust exist, workinfo: %+v, msg: %s", *work, err)
			return err
		}
		if exist {
			log.Debugf("[BookmarkFetchWorker] Skip exist illust, workinfo: %+v", *work)
		} else {
			log.Infof("[BookmarkFetchWorker] Success get bookmark work, workinfo: %+v", *work)
			pbw.workChan <- work
		}
	}
	return nil
}

type IllustInfoFetchWorker struct {
	Worker
	workChan   <-chan *BookmarkWork
	illustChan chan<- *Illust
}

func NewIllustInfoFetchWorker(conf *Config, illustMgr IllustInfoManager, workChan <-chan *BookmarkWork, illustChan chan<- *Illust) IllustInfoFetchWorker {
	return IllustInfoFetchWorker{
		Worker: Worker{
			conf:      conf,
			illustMgr: illustMgr,
			client:    NewPixivClient(conf.Cookie, conf.UserAgent, conf.ParseTimeoutMs),
		},
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

func (w *IllustInfoFetchWorker) processIllustInfo(work *BookmarkWork) {
	w.retry(func() bool {
		illusts, err := w.client.GetIllustInfo(work.Id)
		if err == ErrNotFound || err == ErrFailedUnmarshal {
			log.Warningf("[IllustInfoFetchWorker] Skip illust, %+v, msg: %s", *work, err)
			return true
		}
		if err != nil {
			log.Warningf("[IllustInfoFetchWorker] Failed to get illust info, %+v , msg: %s", *work, err)
			return false
		}
		log.Infof("[IllustInfoFetchWorker] Success get illust, %s", illusts[0].DescriptionString())
		for idx := range illusts {
			illustP := illusts[idx]
			w.illustChan <- illustP
		}
		return true
	})
}

type IllustDownloadWorker struct {
	Worker
	illustChan <-chan *Illust
}

func NewIllustDownloadWorker(conf *Config, illustMgr IllustInfoManager, illustChan <-chan *Illust) IllustDownloadWorker {
	return IllustDownloadWorker{
		Worker: Worker{
			conf:      conf,
			illustMgr: illustMgr,
			client:    NewPixivClient(conf.Cookie, conf.UserAgent, conf.DownloadTimeoutMs),
		},
		illustChan: illustChan,
	}
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

func FormatFileName(illust *Illust, pattern string) string {
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

func (w *IllustDownloadWorker) saveIllustInfo(illust *Illust, data []byte, fileName string) error {
	return Retry(func() error {
		return w.illustMgr.SaveIllust(illust, fmt.Sprintf("%x", sha1.Sum(data)), fileName)
	}, 3)
}

func (w *IllustDownloadWorker) DownloadIllust(illust *Illust) {
	fileName := FormatFileName(illust, w.conf.FileNamePattern)
	fullFileName := filepath.Join(w.conf.DownloadPath, fileName)
	w.retry(func() bool {
		data, err := w.client.DownloadIllust(illust.Urls.Original)
		if err == ErrNotFound || err == ErrFailedUnmarshal {
			return true
		}
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to download illust, retry, %s, url: %s, msg: %s", illust.DescriptionString(), illust.Urls.Original, err)
			return false
		}

		err = w.writeFile(fullFileName, data)
		if err != nil {
			log.Warningf("[IllustDownloadWorker] Failed to write illust, retry, %s, url: %s, msg: %s", illust.DescriptionString(), illust.Urls.Original, err)
			return false
		}

		err = w.saveIllustInfo(illust, data, fileName)
		if err != nil {
			log.Errorf("[IllustDownloadWorker] Failed to write DB, retry, %s, msg: %s", illust.DescriptionString(), err)
			return false
		}
		log.Infof("[IllustDownloadWorker] Success download illust, %s, url: %s", illust.DescriptionString(), illust.Urls.Original)
		return true
	})
}

func Start(conf *Config, illustMgr IllustInfoManager) {
	workChan := make(chan *BookmarkWork, 100)
	illustChan := make(chan *Illust, 100)

	bookmarksFetchWorker := NewPixivBookmarksWorker(conf, illustMgr, workChan)
	bookmarksFetchWorker.Run()

	illustFetchWorker := NewIllustInfoFetchWorker(conf, illustMgr, workChan, illustChan)
	illustFetchWorker.Run()

	illustDownloadWorker := NewIllustDownloadWorker(conf, illustMgr, illustChan)
	illustDownloadWorker.Run()
}
