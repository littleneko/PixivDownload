package app

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	set "github.com/deckarep/golang-set/v2"
	log "github.com/sirupsen/logrus"
)

type DownloadScopeType int

const (
	DownloadScopeInvalid DownloadScopeType = -1
	DownloadScopeAll     DownloadScopeType = iota
	DownloadScopeBookmarks
	DownloadScopeFollowing
	DownloadScopeUsers
	DownloadScopeIllusts
)

var downloadScopeTypes = func() map[string]DownloadScopeType {
	return map[string]DownloadScopeType{
		"ALL":       DownloadScopeAll,
		"BOOKMARKS": DownloadScopeBookmarks,
		"FOLLOWING": DownloadScopeFollowing,
		"USERS":     DownloadScopeUsers,
		"ILLUSTS":   DownloadScopeIllusts,
	}
}

func GetDownloadScopeType(typeStr string) DownloadScopeType {
	typeStrLower := strings.ToUpper(typeStr)
	t, ok := downloadScopeTypes()[typeStrLower]
	if !ok {
		return DownloadScopeInvalid
	}
	return t
}

func ParseDownloadScope(scopes []string) (set.Set[DownloadScopeType], error) {
	ret := set.NewThreadUnsafeSet[DownloadScopeType]()
	for _, s := range scopes {
		ds := GetDownloadScopeType(s)
		if ds == DownloadScopeInvalid {
			return nil, errors.New(fmt.Sprintf("unknown download scope '%s'", s))
		}
		ret.Add(ds)
	}
	return ret, nil
}

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
	Worker

	workChan chan<- *BasicIllustInfo
	offset   int32
	total    int32

	userWhiteListFilter map[PixivID]struct{}
	userBlockListFilter map[PixivID]struct{}
}

func NewPixivBookmarksWorker(options *PixivDlOptions, illustMgr IllustInfoManager, workChan chan<- *BasicIllustInfo) *BookmarksWorker {
	worker := &BookmarksWorker{
		Worker: Worker{
			options:   options,
			illustMgr: illustMgr,
			client:    NewPixivClient(options.Cookie, options.UserAgent, options.ParseTimeoutMs),
		},
		workChan:            workChan,
		offset:              0,
		total:               -1,
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
		if !pbw.hasMorePage() {
			log.Infof("[BookmarkFetchWorker] End scan all bookmarks, wait for next round")
			pbw.offset = 0
			pbw.total = -1
			time.Sleep(time.Duration(pbw.options.ScanIntervalSec) * time.Second)
		}
		pbw.retry(func() bool {
			bmBody, err := pbw.client.GetBookmarks(pbw.options.UserId, pbw.offset, BookmarksPageLimit)
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
			log.Infof("[BookmarkFetchWorker] Success get bookmarks, offset: %d", pbw.offset)
			return true
		})
		pbw.moveToNextPage()
	}
}

func (pbw *BookmarksWorker) moveToNextPage() {
	pbw.offset += BookmarksPageLimit
}

func (pbw *BookmarksWorker) hasMorePage() bool {
	return pbw.total == -1 || pbw.offset < pbw.total
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

func (pbw *BookmarksWorker) writeToQueue(bmBody *BookmarksBody) error {
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
	Worker
	workChan   <-chan *BasicIllustInfo
	illustChan chan<- *IllustInfo
}

func NewIllustInfoFetchWorker(options *PixivDlOptions, illustMgr IllustInfoManager, workChan <-chan *BasicIllustInfo, illustChan chan<- *IllustInfo) *IllustInfoFetchWorker {
	worker := &IllustInfoFetchWorker{
		Worker: Worker{
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
	Worker
	illustChan <-chan *IllustInfo
}

func NewIllustDownloadWorker(options *PixivDlOptions, illustMgr IllustInfoManager, illustChan <-chan *IllustInfo) *IllustDownloadWorker {
	worker := &IllustDownloadWorker{
		Worker: Worker{
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
		log.Infof("[IllustDownloadWorker] Success download illust: %s, filename: %s, url: %s", illust.DigestString(), fullFileName, illust.Urls.Original)
		return true
	})
}

func Start(options *PixivDlOptions, illustMgr IllustInfoManager) error {
	downloadScopes, err := ParseDownloadScope(options.DownloadScope)
	if err != nil {
		return err
	}

	workChan := make(chan *BasicIllustInfo, 100)
	illustChan := make(chan *IllustInfo, 100)

	if (downloadScopes.Contains(DownloadScopeAll) || downloadScopes.Contains(DownloadScopeIllusts)) &&
		len(options.DownloadIllustIds) > 0 {
		go func() {
			for _, pid := range options.DownloadIllustIds {
				workChan <- &BasicIllustInfo{
					Id: PixivID(pid),
				}
			}
		}()
	}
	if (downloadScopes.Contains(DownloadScopeAll) || downloadScopes.Contains(DownloadScopeBookmarks)) &&
		len(options.UserId) > 0 {
		bookmarksFetchWorker := NewPixivBookmarksWorker(options, illustMgr, workChan)
		bookmarksFetchWorker.Run()
	}

	illustFetchWorker := NewIllustInfoFetchWorker(options, illustMgr, workChan, illustChan)
	illustFetchWorker.Run()

	illustDownloadWorker := NewIllustDownloadWorker(options, illustMgr, illustChan)
	illustDownloadWorker.Run()

	return nil
}
