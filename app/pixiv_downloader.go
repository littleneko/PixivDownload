package app

import (
	"time"

	pixiv "github.com/littleneko/pixiv-api-go"
	log "github.com/sirupsen/logrus"
)

type PixivDownloader interface {
	Start()
	Close()
}

// IllustDownloader download the illust by pid
type IllustDownloader struct {
	illustInfoWorker     *IllustInfoWorker
	illustDownloadWorker *IllustDownloadWorker

	options *PixivDlOptions

	basicIllustChan chan *pixiv.IllustDigest
	fullIllustChan  chan *pixiv.IllustInfo
}

func NewIllustDownloader(options *PixivDlOptions, illustMgr IllustInfoManager) *IllustDownloader {
	basicIllustChan := make(chan *pixiv.IllustDigest, 50)
	fullIllustChan := make(chan *pixiv.IllustInfo, 100)

	downloader := &IllustDownloader{
		illustInfoWorker:     NewIllustInfoWorker(options, illustMgr, basicIllustChan, fullIllustChan),
		illustDownloadWorker: NewIllustDownloadWorker(options, illustMgr, fullIllustChan),
		options:              options,
		basicIllustChan:      basicIllustChan,
		fullIllustChan:       fullIllustChan,
	}
	return downloader
}

func (d *IllustDownloader) waitDone(illustCnt uint64) {
	for {
		if d.illustInfoWorker.GetConsumeCnt() == illustCnt &&
			d.illustDownloadWorker.GetConsumeCnt() == d.illustInfoWorker.GetProduceCnt() {
			d.illustInfoWorker.ResetCnt()
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

	d.illustInfoWorker.Run()
	d.illustDownloadWorker.Run()

	for {
		for _, pid := range d.options.DownloadIllustIds {
			d.basicIllustChan <- &pixiv.IllustDigest{
				Id:        pixiv.PixivID(pid),
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
	bookmarksWorker      *BookmarksWorker
	illustInfoWorker     *IllustInfoWorker
	illustDownloadWorker *IllustDownloadWorker

	options *PixivDlOptions

	uidChan         chan pixiv.PixivID
	basicIllustChan chan *pixiv.IllustDigest
	fullIllustChan  chan *pixiv.IllustInfo
}

func NewBookmarksDownloader(options *PixivDlOptions, illustMgr IllustInfoManager) *BookmarksDownloader {
	uidChan := make(chan pixiv.PixivID, 10)
	basicIllustChan := make(chan *pixiv.IllustDigest, 50)
	fullIllustChan := make(chan *pixiv.IllustInfo, 100)

	downloader := &BookmarksDownloader{
		bookmarksWorker:      NewBookmarksWorker(options, illustMgr, uidChan, basicIllustChan),
		illustInfoWorker:     NewIllustInfoWorker(options, illustMgr, basicIllustChan, fullIllustChan),
		illustDownloadWorker: NewIllustDownloadWorker(options, illustMgr, fullIllustChan),
		options:              options,
		uidChan:              uidChan,
		basicIllustChan:      basicIllustChan,
		fullIllustChan:       fullIllustChan,
	}
	return downloader
}

func (d *BookmarksDownloader) waitDone(userCnt uint64) {
	for {
		if d.bookmarksWorker.GetConsumeCnt() == userCnt &&
			d.illustInfoWorker.GetConsumeCnt() == d.bookmarksWorker.GetProduceCnt() &&
			d.illustDownloadWorker.GetConsumeCnt() == d.illustInfoWorker.GetProduceCnt() {
			d.bookmarksWorker.ResetCnt()
			d.illustInfoWorker.ResetCnt()
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
	d.illustInfoWorker.Run()
	d.illustDownloadWorker.Run()

	for {
		for _, uid := range d.options.DownloadBookmarksUserIds {
			d.uidChan <- pixiv.PixivID(uid)
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
	artistWorker         *ArtistWorker
	illustInfoWorker     *IllustInfoWorker
	illustDownloadWorker *IllustDownloadWorker

	options *PixivDlOptions

	uidChan         chan pixiv.PixivID
	basicIllustChan chan *pixiv.IllustDigest
	fullIllustChan  chan *pixiv.IllustInfo
}

func NewArtistDownloader(options *PixivDlOptions, illustMgr IllustInfoManager) *ArtistDownloader {
	uidChan := make(chan pixiv.PixivID, 10)
	basicIllustChan := make(chan *pixiv.IllustDigest, 50)
	fullIllustChan := make(chan *pixiv.IllustInfo, 100)

	downloader := &ArtistDownloader{
		artistWorker:         NewArtistWorker(options, illustMgr, uidChan, basicIllustChan),
		illustInfoWorker:     NewIllustInfoWorker(options, illustMgr, basicIllustChan, fullIllustChan),
		illustDownloadWorker: NewIllustDownloadWorker(options, illustMgr, fullIllustChan),
		options:              options,
		uidChan:              uidChan,
		basicIllustChan:      basicIllustChan,
		fullIllustChan:       fullIllustChan,
	}
	return downloader
}

func (d *ArtistDownloader) waitDone(userCnt uint64) {
	for {
		if d.artistWorker.GetConsumeCnt() == userCnt &&
			d.illustInfoWorker.GetConsumeCnt() == d.artistWorker.GetProduceCnt() &&
			d.illustDownloadWorker.GetConsumeCnt() == d.illustInfoWorker.GetProduceCnt() {
			d.artistWorker.ResetCnt()
			d.illustInfoWorker.ResetCnt()
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
	d.illustInfoWorker.Run()
	d.illustDownloadWorker.Run()

	for {
		for _, uid := range d.options.DownloadArtistUserIds {
			d.uidChan <- pixiv.PixivID(uid)
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
