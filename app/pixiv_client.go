package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var (
	ErrNotFound        = errors.New("NotFound")
	ErrFailedUnmarshal = errors.New("FailedUnmarshal")
)

const (
	bookmarksUrl     = "https://www.pixiv.net/ajax/user/%s/illusts/bookmarks"
	followingUrl     = "https://www.pixiv.net/ajax/user/%s/following"
	illustInfoUrl    = "https://www.pixiv.net/ajax/illust/%s"
	illustPagesUrl   = "https://www.pixiv.net/ajax/illust/%s/pages"
	userAllIllustUrl = "https://www.pixiv.net/ajax/user/%s/profile/all"
)

const (
	bookmarksReferUrl      = "https://www.pixiv.net/users/%s/bookmarks/artworks"
	followingReferUrl      = "https://www.pixiv.net/users/%s/following"
	illustInfoReferUrl     = "https://www.pixiv.net/artworks/%s"
	illustDownloadReferUrl = "https://www.pixiv.net"
	userAllIllustReferUrl  = "https://www.pixiv.net/users/%s"
)

type pageUrlType int

const (
	pageUrlTypeBookmarks pageUrlType = iota
	pageUrlTypeFollowing
)

func genPageUrl(uid string, offset, limit int32, urlType pageUrlType) (string, error) {
	params := url.Values{}
	params.Set("tag", "")
	params.Set("offset", strconv.FormatInt(int64(offset), 10))
	params.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.Set("rest", "show")

	var pUrl *url.URL
	switch urlType {
	case pageUrlTypeBookmarks:
		pUrl, _ = url.Parse(fmt.Sprintf(bookmarksUrl, uid))
		break
	case pageUrlTypeFollowing:
		pUrl, _ = url.Parse(fmt.Sprintf(followingUrl, uid))
		break
	default:
		return "", errors.New("unknown page type")
	}
	pUrl.RawQuery = params.Encode()
	return pUrl.String(), nil
}

type PixivClient struct {
	client *http.Client
	header map[string]string
}

func NewPixivClient(cookie string, userAgent string, timeoutMs int32) *PixivClient {
	header := map[string]string{
		"Cookie":     cookie,
		"User-Agent": userAgent,
	}
	return NewPixivClientWithHeader(header, timeoutMs)
}

func NewPixivClientWithHeader(header map[string]string, timeoutMs int32) *PixivClient {
	pc := &PixivClient{
		client: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
		header: header,
	}
	return pc
}

func (p *PixivClient) getRaw(url string, refer string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Referer", refer)
	for k, v := range p.header {
		req.Header.Add(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == 404 {
		return resp, ErrNotFound
	}
	if resp.StatusCode != 200 {
		return resp, errors.New(fmt.Sprintf("code: %d, message: %s", resp.StatusCode, resp.Status))
	}
	return resp, nil
}

func (p *PixivClient) getPixivResp(url string, refer string) (*PixivResponse, error) {
	resp, err := p.getRaw(url, refer)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var jResp PixivResponse
	_ = json.Unmarshal(body, &jResp)
	if jResp.Error {
		return nil, errors.New(fmt.Sprintf("Pixiv response error: %s", jResp.Message))
	}

	return &jResp, nil
}

func (p *PixivClient) GetBookmarks(uid string, offset, limit int32) (*BookmarksInfo, error) {
	bUrl, err := genPageUrl(uid, offset, limit, pageUrlTypeBookmarks)
	if err != nil {
		return nil, err
	}
	refer := fmt.Sprintf(bookmarksReferUrl, uid)
	resp, err := p.getPixivResp(bUrl, refer)
	if err != nil {
		return nil, err
	}

	var bookmarks BookmarksInfo
	err = json.Unmarshal(resp.Body, &bookmarks)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}
	return &bookmarks, nil
}

func (p *PixivClient) GetFollowing(uid string, offset, limit int32) (*FollowingInfo, error) {
	bUrl, err := genPageUrl(uid, offset, limit, pageUrlTypeFollowing)
	if err != nil {
		return nil, err
	}
	refer := fmt.Sprintf(followingReferUrl, uid)
	resp, err := p.getPixivResp(bUrl, refer)
	if err != nil {
		return nil, err
	}

	var following FollowingInfo
	err = json.Unmarshal(resp.Body, &following)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}
	return &following, nil
}

func (p *PixivClient) GetUserIllusts(uid string) ([]*BasicIllustInfo, error) {
	allUrl := fmt.Sprintf(userAllIllustUrl, uid)
	refer := fmt.Sprintf(userAllIllustReferUrl, uid)
	resp, err := p.getPixivResp(allUrl, refer)
	if err != nil {
		return nil, err
	}

	var body struct {
		Illusts map[string]struct{} `json:"illusts"`
	}
	err = json.Unmarshal(resp.Body, &body)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}

	illusts := make([]*BasicIllustInfo, len(body.Illusts))
	for k, _ := range body.Illusts {
		illusts = append(illusts, &BasicIllustInfo{
			Id: PixivID(k),
		})
	}
	return illusts, nil
}

func (p *PixivClient) GetIllustInfo(illustId PixivID, onlyP0 bool) ([]*IllustInfo, error) {
	illust, err := p.getIllustBasicInfo(illustId)
	if err != nil {
		return nil, err
	}
	if illust.PageCount == 1 || onlyP0 {
		return []*IllustInfo{illust}, nil
	} else {
		return p.getIllustAllPages(illust)
	}
}

func (p *PixivClient) getIllustBasicInfo(illustId PixivID) (*IllustInfo, error) {
	illustUrl := fmt.Sprintf(illustInfoUrl, illustId)
	refer := fmt.Sprintf(illustInfoReferUrl, illustId)
	iResp, err := p.getPixivResp(illustUrl, refer)
	if err != nil {
		return nil, err
	}

	var illust struct {
		*IllustInfo
		RawTags json.RawMessage `json:"tags"`
	}
	err = json.Unmarshal(iResp.Body, &illust)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}

	/**
	The json format of tags:

	"tags": {
	            "authorId": "3494650",
	            "isLocked": false,
	            "tags": [
	                {
	                    "tag": "R-18",
	                    "locked": true,
	                    "deletable": false,
	                    "userId": "3494650",
	                    "userName": "はすね"
	                },
	                {
	                    "tag": "小悪魔",
	                    "locked": true,
	                    "deletable": false,
	                    "userId": "3494650",
	                    "translation": {
	                        "en": "小恶魔"
	                    },
	                    "userName": "はすね"
	                },
	            ],
	            "writable": true
	        },
	*/

	var tags struct {
		Tags []struct {
			Tag string `json:"tag"`
		} `json:"tags"`
	}
	err = json.Unmarshal(illust.RawTags, &tags)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}

	r18 := false
	for _, tag := range tags.Tags {
		if tag.Tag == "R-18" {
			r18 = true
		}
		illust.Tags = append(illust.Tags, tag.Tag)
	}
	illust.R18 = r18

	return illust.IllustInfo, nil
}

func (p *PixivClient) getIllustAllPages(seed *IllustInfo) ([]*IllustInfo, error) {
	illustUrl := fmt.Sprintf(illustPagesUrl, seed.Id)
	refer := fmt.Sprintf(illustInfoReferUrl, seed.Id)
	iResp, err := p.getPixivResp(illustUrl, refer)
	if err != nil {
		return nil, err
	}

	type IllustPagesUnit struct {
		Urls Urls `json:"urls"`
	}
	var illustPageBody []IllustPagesUnit
	err = json.Unmarshal(iResp.Body, &illustPageBody)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}

	var illusts []*IllustInfo
	for idx := range illustPageBody {
		illust := *seed
		illust.CurPage = idx
		illust.Urls = illustPageBody[idx].Urls
		illusts = append(illusts, &illust)
	}
	return illusts, nil
}

func (p *PixivClient) getIllustData(url string) ([]byte, error) {
	resp, err := p.getRaw(url, illustDownloadReferUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type BookmarksFetcher struct {
	client    *PixivClient
	uid       string
	limit     int32
	total     int32
	curOffset int32
}

func NewBookmarksFetcher(client *PixivClient, uid string, limit int32) *BookmarksFetcher {
	return &BookmarksFetcher{
		client:    client,
		uid:       uid,
		limit:     limit,
		total:     -1,
		curOffset: 0,
	}
}
func (bf *BookmarksFetcher) CurOffset() int32 {
	return bf.curOffset
}

func (bf *BookmarksFetcher) MoveToNextPage() {
	bf.curOffset += bf.limit
}

func (bf *BookmarksFetcher) HasMorePage() bool {
	return bf.total == -1 || bf.curOffset < bf.total
}

func (bf *BookmarksFetcher) GetNextPageBookmarks() (*BookmarksInfo, error) {
	bmInfo, err := bf.client.GetBookmarks(bf.uid, bf.curOffset, bf.limit)
	if err != nil {
		return nil, err
	}
	if bmInfo.Total > 0 {
		bf.total = bmInfo.Total
	}
	return bmInfo, nil
}
