package app

import (
	log "github.com/sirupsen/logrus"
	"time"
)

type PixivDownloader interface {
	Start()
	Close()
}

// IllustDownloader download the illust by pid
type IllustDownloader struct {
	illustInfoFetchWorker *IllustInfoFetchWorker
	illustDownloadWorker  *IllustDownloadWorker

	options *PixivDlOptions

	basicIllustChan chan *BasicIllustInfo
	fullIllustChan  chan *FullIllustInfo
}

func NewIllustDownloader(options *PixivDlOptions, illustMgr IllustInfoManager) *IllustDownloader {
	basicIllustChan := make(chan *BasicIllustInfo, 50)
	fullIllustChan := make(chan *FullIllustInfo, 100)

	downloader := &IllustDownloader{
		illustInfoFetchWorker: NewIllustInfoFetchWorker(options, illustMgr, basicIllustChan, fullIllustChan),
		illustDownloadWorker:  NewIllustDownloadWorker(options, illustMgr, fullIllustChan),
		options:               options,
		basicIllustChan:       basicIllustChan,
		fullIllustChan:        fullIllustChan,
	}
	return downloader
}

func (d *IllustDownloader) waitDone(illustCnt uint64) {
	for {
		if d.illustInfoFetchWorker.GetConsumeCnt() == illustCnt &&
			d.illustDownloadWorker.GetConsumeCnt() == d.illustInfoFetchWorker.GetProduceCnt() {
			d.illustInfoFetchWorker.ResetCnt()
			d.illustDownloadWorker.ResetCnt()
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func (d *IllustDownloader) Start() {
	if len(d.options.DownloadIllustIds) == 0 {
		return
	}

	d.illustInfoFetchWorker.Run()
	d.illustDownloadWorker.Run()

	for {
		for _, pid := range d.options.DownloadIllustIds {
			d.basicIllustChan <- &BasicIllustInfo{
				Id:        PixivID(pid),
				PageCount: 1,
			}
		}

		d.waitDone(uint64(len(d.options.DownloadIllustIds)))
		if !d.options.ServiceMode {
			break
		}
		duration := time.Duration(d.options.ScanIntervalSec) * time.Second
		log.Infof("[IllustDownloader] wait for next round after %s", duration)
		time.Sleep(duration)
	}
}

func (d *IllustDownloader) Close() {
	close(d.basicIllustChan)
	close(d.fullIllustChan)
}

// BookmarksDownloader download the illust of users bookmarks
type BookmarksDownloader struct {
	bookmarksWorker       *BookmarksWorker
	illustInfoFetchWorker *IllustInfoFetchWorker
	illustDownloadWorker  *IllustDownloadWorker

	options *PixivDlOptions

	uidChan         chan PixivID
	basicIllustChan chan *BasicIllustInfo
	fullIllustChan  chan *FullIllustInfo
}

func NewBookmarksDownloader(options *PixivDlOptions, illustMgr IllustInfoManager) *BookmarksDownloader {
	uidChan := make(chan PixivID, 10)
	basicIllustChan := make(chan *BasicIllustInfo, 50)
	fullIllustChan := make(chan *FullIllustInfo, 100)

	downloader := &BookmarksDownloader{
		bookmarksWorker:       NewBookmarksWorker(options, illustMgr, uidChan, basicIllustChan),
		illustInfoFetchWorker: NewIllustInfoFetchWorker(options, illustMgr, basicIllustChan, fullIllustChan),
		illustDownloadWorker:  NewIllustDownloadWorker(options, illustMgr, fullIllustChan),
		options:               options,
		uidChan:               uidChan,
		basicIllustChan:       basicIllustChan,
		fullIllustChan:        fullIllustChan,
	}
	return downloader
}

func (d *BookmarksDownloader) waitDone(userCnt uint64) {
	for {
		if d.bookmarksWorker.GetConsumeCnt() == userCnt &&
			d.illustInfoFetchWorker.GetConsumeCnt() == d.bookmarksWorker.GetProduceCnt() &&
			d.illustDownloadWorker.GetConsumeCnt() == d.illustInfoFetchWorker.GetProduceCnt() {
			d.bookmarksWorker.ResetCnt()
			d.illustInfoFetchWorker.ResetCnt()
			d.illustDownloadWorker.ResetCnt()
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func (d *BookmarksDownloader) Start() {
	if len(d.options.DownloadBookmarksUserIds) == 0 {
		return
	}

	d.bookmarksWorker.Run()
	d.illustInfoFetchWorker.Run()
	d.illustDownloadWorker.Run()

	for {
		for _, uid := range d.options.DownloadBookmarksUserIds {
			d.uidChan <- PixivID(uid)
		}

		d.waitDone(uint64(len(d.options.DownloadBookmarksUserIds)))
		if !d.options.ServiceMode {
			break
		}
		duration := time.Duration(d.options.ScanIntervalSec) * time.Second
		log.Infof("[BookmarksDownloader] wait for next round after %s", duration)
		time.Sleep(duration)
	}
}

func (d *BookmarksDownloader) Close() {
	close(d.uidChan)
	close(d.basicIllustChan)
	close(d.fullIllustChan)
}

// ArtistDownloader download all the illust of users
type ArtistDownloader struct {
	artistWorker          *ArtistWorker
	illustInfoFetchWorker *IllustInfoFetchWorker
	illustDownloadWorker  *IllustDownloadWorker

	options *PixivDlOptions

	uidChan         chan PixivID
	basicIllustChan chan *BasicIllustInfo
	fullIllustChan  chan *FullIllustInfo
}

func NewArtistDownloader(options *PixivDlOptions, illustMgr IllustInfoManager) *ArtistDownloader {
	uidChan := make(chan PixivID, 10)
	basicIllustChan := make(chan *BasicIllustInfo, 50)
	fullIllustChan := make(chan *FullIllustInfo, 100)

	downloader := &ArtistDownloader{
		artistWorker:          NewArtistWorker(options, illustMgr, uidChan, basicIllustChan),
		illustInfoFetchWorker: NewIllustInfoFetchWorker(options, illustMgr, basicIllustChan, fullIllustChan),
		illustDownloadWorker:  NewIllustDownloadWorker(options, illustMgr, fullIllustChan),
		options:               options,
		uidChan:               uidChan,
		basicIllustChan:       basicIllustChan,
		fullIllustChan:        fullIllustChan,
	}
	return downloader
}

func (d *ArtistDownloader) waitDone(userCnt uint64) {
	for {
		if d.artistWorker.GetConsumeCnt() == userCnt &&
			d.illustInfoFetchWorker.GetConsumeCnt() == d.artistWorker.GetProduceCnt() &&
			d.illustDownloadWorker.GetConsumeCnt() == d.illustInfoFetchWorker.GetProduceCnt() {
			d.artistWorker.ResetCnt()
			d.illustInfoFetchWorker.ResetCnt()
			d.illustDownloadWorker.ResetCnt()
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func (d *ArtistDownloader) Start() {
	if len(d.options.DownloadArtistUserIds) == 0 {
		return
	}

	d.artistWorker.Run()
	d.illustInfoFetchWorker.Run()
	d.illustDownloadWorker.Run()

	for {
		for _, uid := range d.options.DownloadArtistUserIds {
			d.uidChan <- PixivID(uid)
		}

		d.waitDone(uint64(len(d.options.DownloadArtistUserIds)))
		if !d.options.ServiceMode {
			break
		}
		duration := time.Duration(d.options.ScanIntervalSec) * time.Second
		log.Infof("[ArtistDownloader] wait for next round after %s", duration)
		time.Sleep(duration)
	}
}

func (d *ArtistDownloader) Close() {
	close(d.uidChan)
	close(d.basicIllustChan)
	close(d.fullIllustChan)
}
