package app

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
	"time"
)

type PixivID string

func (w *PixivID) UnmarshalJSON(data []byte) (err error) {
	if zip, err := strconv.Atoi(string(data)); err == nil {
		str := strconv.Itoa(zip)
		*w = PixivID(str)
		return nil
	}
	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), w)
}

type PixivResponse struct {
	Error   bool            `json:"error"`
	Message string          `json:"message"`
	Body    json.RawMessage `json:"body"`
}

type UserInfo struct {
	UserId      PixivID `json:"userId"`
	UserName    string  `json:"userName"`
	UserAccount string  `json:"userAccount"`
}

// IllustDigest is the illust basic info get from bookmarks or artist work
type IllustDigest struct {
	Id        PixivID `json:"id"`
	Title     string  `json:"title"`
	PageCount int32   `json:"pageCount"`
	UserInfo
}

func (bi *IllustDigest) DigestString() string {
	return fmt.Sprintf("[id: %s, title: %s, uid: %s, uname: %s, pages: %d]", bi.Id, bi.Title, bi.UserId, bi.UserName, bi.PageCount)
}

type BookmarksInfo struct {
	Total int32          `json:"total"`
	Works []IllustDigest `json:"works"`
}

type FollowingInfo struct {
	Users []UserInfo `json:"users"`
	Total int32      `json:"total"`
}

type Urls struct {
	Mini     string `json:"mini"`
	Thumb    string `json:"thumb"`
	Small    string `json:"small"`
	Regular  string `json:"regular"`
	Original string `json:"original"`
}

type IllustInfo struct {
	Id            PixivID   `json:"id"`
	PageIdx       int       `json:"curPage"`
	Title         string    `json:"title"`
	Urls          Urls      `json:"urls"`
	R18           bool      `json:"r18,omitempty"`
	Tags          []string  `json:"tags,omitempty"`
	Description   string    `json:"description"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	PageCount     int       `json:"pageCount"`
	BookmarkCount int       `json:"bookmarkCount"`
	LikeCount     int       `json:"likeCount"`
	CommentCount  int       `json:"commentCount"`
	ViewCount     int       `json:"viewCount"`
	CreateDate    time.Time `json:"createDate"`
	UploadDate    time.Time `json:"uploadDate"`
	UserInfo
}

func (i *IllustInfo) DigestString() string {
	return fmt.Sprintf("[id: %s, page: %d, title: %s, uid: %s, uname: %s, pageCnt: %d, R18: %v, bookmarkCnt: %d, likeCnt: %d]",
		i.Id, i.PageIdx, i.Title, i.UserId, i.UserName, i.PageCount, i.R18, i.BookmarkCount, i.LikeCount)
}

func (i *IllustInfo) DigestStringWithUrl() string {
	return fmt.Sprintf("[id: %s, page: %d, title: %s, uid: %s, uname: %s, pageCnt: %d, R18: %v, bookmarkCnt: %d, likeCnt: %d, width: %d, height: %d, URL: %s]",
		i.Id, i.PageIdx, i.Title, i.UserId, i.UserName, i.PageCount, i.R18, i.BookmarkCount, i.LikeCount, i.Width, i.Height, i.Urls.Original)
}

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

func NewPixivClient(timeoutMs int32) *PixivClient {
	return NewPixivClientWithProxy(nil, timeoutMs)
}

func NewPixivClientWithProxy(proxy *url.URL, timeoutMs int32) *PixivClient {
	var tr *http.Transport
	if proxy != nil {
		tr = &http.Transport{Proxy: http.ProxyURL(proxy)}
	} else {
		tr = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	pc := &PixivClient{
		client: &http.Client{
			Timeout:   time.Duration(timeoutMs) * time.Millisecond,
			Transport: tr,
		},
		header: map[string]string{},
	}
	return pc
}

func (p *PixivClient) SetHeader(header map[string]string) {
	p.header = header
}

func (p *PixivClient) AddHeader(key, value string) {
	p.header[key] = value
}

func (p *PixivClient) SetCookie(value string) {
	p.AddHeader("Cookie", value)
}

func (p *PixivClient) SetUserAgent(value string) {
	p.AddHeader("User-Agent", value)
}

func (p *PixivClient) getRaw(url, refer string) (*http.Response, error) {
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

func (p *PixivClient) getPixivResp(url, refer string) (*PixivResponse, error) {
	resp, err := p.getRaw(url, refer)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

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

// GetUserBookmarks get the bookmarks of a user
func (p *PixivClient) GetUserBookmarks(uid string, offset, limit int32) (*BookmarksInfo, error) {
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

// GetUserFollowing get the following of a user
func (p *PixivClient) GetUserFollowing(uid string, offset, limit int32) (*FollowingInfo, error) {
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

// GetUserIllusts get all illusts of the user
func (p *PixivClient) GetUserIllusts(uid string) ([]PixivID, error) {
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
		// if the user has no illust, the json value type is list?
		var body struct {
			Illusts []PixivID `json:"illusts"`
		}
		err = json.Unmarshal(resp.Body, &body)
		if err != nil {
			return nil, ErrFailedUnmarshal
		}
		return body.Illusts, nil
	}

	illusts := make([]PixivID, 0, len(body.Illusts))
	for k, _ := range body.Illusts {
		illusts = append(illusts, PixivID(k))
	}
	return illusts, nil
}

// GetIllustInfo get the illust detail for the illust id. For a multi page illust,
// only the first page will be get if onlyP0 is true.
func (p *PixivClient) GetIllustInfo(illustId PixivID, onlyP0 bool) ([]*IllustInfo, error) {
	illust, err := p.getBasicIllustInfo(illustId)
	if err != nil {
		return nil, err
	}
	if illust.PageCount == 1 || onlyP0 {
		return []*IllustInfo{illust}, nil
	} else {
		return p.getMultiPagesIllustInfo(illust)
	}
}

func (p *PixivClient) getBasicIllustInfo(illustId PixivID) (*IllustInfo, error) {
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
	                    "userName": "?????????"
	                },
	                {
	                    "tag": "?????????",
	                    "locked": true,
	                    "deletable": false,
	                    "userId": "3494650",
	                    "translation": {
	                        "en": "?????????"
	                    },
	                    "userName": "?????????"
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

func (p *PixivClient) getMultiPagesIllustInfo(seed *IllustInfo) ([]*IllustInfo, error) {
	illustUrl := fmt.Sprintf(illustPagesUrl, seed.Id)
	refer := fmt.Sprintf(illustInfoReferUrl, seed.Id)
	iResp, err := p.getPixivResp(illustUrl, refer)
	if err != nil {
		return nil, err
	}

	type IllustPagesUnit struct {
		Urls   Urls `json:"urls"`
		Width  int  `json:"width"`
		Height int  `json:"height"`
	}
	var illustPageBody []IllustPagesUnit
	err = json.Unmarshal(iResp.Body, &illustPageBody)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}

	var illusts []*IllustInfo
	for idx := range illustPageBody {
		illust := *seed
		illust.PageIdx = idx
		illust.Urls = illustPageBody[idx].Urls
		illust.Width = illustPageBody[idx].Width
		illust.Height = illustPageBody[idx].Height
		illusts = append(illusts, &illust)
	}
	return illusts, nil
}

func (p *PixivClient) GetIllust(url string) (io.ReadCloser, error) {
	resp, err := p.getRaw(url, illustDownloadReferUrl)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// GetIllustData return all the illust bytes, may be OOM
func (p *PixivClient) GetIllustData(url string) ([]byte, error) {
	resp, err := p.GetIllust(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Close()
	}()

	data, err := io.ReadAll(resp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// DownloadIllust download the illust to filename, return the file size and sha1 sum
func (p *PixivClient) DownloadIllust(url, filename string) (int64, string, error) {
	dirName := filepath.Dir(filename)
	err := CheckAndMkdir(dirName)
	if err != nil {
		return 0, "", err
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_ = file.Close()
	}()

	resp, err := p.getRaw(url, illustDownloadReferUrl)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	h := sha1.New()
	r := io.TeeReader(resp.Body, h)
	size, err := io.Copy(file, r)
	sum := fmt.Sprintf("%x", h.Sum(nil))

	return size, sum, err
}

type pixivPageClient struct {
	client    *PixivClient
	uid       string
	limit     int32
	total     int32
	curOffset int32
}

func (pc *pixivPageClient) CurOffset() int32 {
	return pc.curOffset
}

func (pc *pixivPageClient) Total() int32 {
	return pc.total
}

func (pc *pixivPageClient) MoveToNextPage() {
	pc.curOffset += pc.limit
}

func (pc *pixivPageClient) HasMorePage() bool {
	return pc.total == -1 || pc.curOffset < pc.total
}

// PixivBookmarksPageClient is a wrapper of PixivClient.GetUserBookmarks, it records the bookmarks page offset and num
type PixivBookmarksPageClient struct {
	pixivPageClient
}

func NewBookmarksPageClient(client *PixivClient, uid string, limit int32) *PixivBookmarksPageClient {
	return &PixivBookmarksPageClient{
		pixivPageClient: pixivPageClient{
			client:    client,
			uid:       uid,
			limit:     limit,
			total:     -1,
			curOffset: 0,
		},
	}
}

func (bpc *PixivBookmarksPageClient) GetNextPageBookmarks() (*BookmarksInfo, error) {
	bmInfo, err := bpc.client.GetUserBookmarks(bpc.uid, bpc.curOffset, bpc.limit)
	// mark this user as invalid user, it has no next page
	if err == ErrNotFound {
		bpc.total = 0
	}
	if err != nil {
		return nil, err
	}
	if bmInfo.Total > bpc.total {
		bpc.total = bmInfo.Total
	}
	return bmInfo, nil
}

// PixivFollowingPageClient is a wrapper of PixivClient.GetUserFollowing, it records the bookmarks page offset and num
type PixivFollowingPageClient struct {
	pixivPageClient
}

func NewFollowingPageClient(client *PixivClient, uid string, limit int32) *PixivFollowingPageClient {
	return &PixivFollowingPageClient{
		pixivPageClient: pixivPageClient{
			client:    client,
			uid:       uid,
			limit:     limit,
			total:     -1,
			curOffset: 0,
		},
	}
}

func (fpc *PixivFollowingPageClient) GetNextPageFollowing() (*FollowingInfo, error) {
	bmInfo, err := fpc.client.GetUserFollowing(fpc.uid, fpc.curOffset, fpc.limit)
	// mark this user as invalid user, it has no next page
	if err == ErrNotFound {
		fpc.total = 0
	}
	if err != nil {
		return nil, err
	}
	if bmInfo.Total > fpc.total {
		fpc.total = bmInfo.Total
	}
	return bmInfo, nil
}
