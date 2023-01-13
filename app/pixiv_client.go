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

// BasicIllustInfo is the illust info get from users bookmarks or users artworks
type BasicIllustInfo struct {
	Id        PixivID `json:"id"`
	Title     string  `json:"title"`
	PageCount int32   `json:"pageCount"`
	UserInfo
}

func (bi *BasicIllustInfo) DigestString() string {
	return fmt.Sprintf("[id: %s, title: %s, uid: %s, uname: %s, pages: %d]", bi.Id, bi.Title, bi.UserId, bi.UserName, bi.PageCount)
}

type BookmarksInfo struct {
	Total int32             `json:"total"`
	Works []BasicIllustInfo `json:"works"`
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

type FullIllustInfo struct {
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

func (i *FullIllustInfo) DigestString() string {
	return fmt.Sprintf("[id: %s, page: %d, title: %s, uid: %s, uname: %s, pageCnt: %d, R18: %v, bookmarkCnt: %d, likeCnt: %d]",
		i.Id, i.PageIdx, i.Title, i.UserId, i.UserName, i.PageCount, i.R18, i.BookmarkCount, i.LikeCount)
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

// GetUserIllusts get all the illust pid of the user
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
		return nil, ErrFailedUnmarshal
	}

	illusts := make([]PixivID, 0, len(body.Illusts))
	for k, _ := range body.Illusts {
		illusts = append(illusts, PixivID(k))
	}
	return illusts, nil
}

func (p *PixivClient) GetIllustInfo(illustId PixivID, onlyP0 bool) ([]*FullIllustInfo, error) {
	illust, err := p.getIllustBasicInfo(illustId)
	if err != nil {
		return nil, err
	}
	if illust.PageCount == 1 || onlyP0 {
		return []*FullIllustInfo{illust}, nil
	} else {
		return p.getIllustAllPages(illust)
	}
}

func (p *PixivClient) getIllustBasicInfo(illustId PixivID) (*FullIllustInfo, error) {
	illustUrl := fmt.Sprintf(illustInfoUrl, illustId)
	refer := fmt.Sprintf(illustInfoReferUrl, illustId)
	iResp, err := p.getPixivResp(illustUrl, refer)
	if err != nil {
		return nil, err
	}

	var illust struct {
		*FullIllustInfo
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

	return illust.FullIllustInfo, nil
}

func (p *PixivClient) getIllustAllPages(seed *FullIllustInfo) ([]*FullIllustInfo, error) {
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

	var illusts []*FullIllustInfo
	for idx := range illustPageBody {
		illust := *seed
		illust.PageIdx = idx
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
	// mark this user as invalid user, it has no next page
	if err == ErrNotFound {
		bf.total = 0
	}
	if err != nil {
		return nil, err
	}
	if bmInfo.Total > 0 {
		bf.total = bmInfo.Total
	}
	return bmInfo, nil
}

type FollowingFetcher struct {
	client    *PixivClient
	uid       string
	limit     int32
	total     int32
	curOffset int32
}

func NewFollowingFetcher(client *PixivClient, uid string, limit int32) *FollowingFetcher {
	return &FollowingFetcher{
		client:    client,
		uid:       uid,
		limit:     limit,
		total:     -1,
		curOffset: 0,
	}
}
func (ff *FollowingFetcher) CurOffset() int32 {
	return ff.curOffset
}

func (ff *FollowingFetcher) MoveToNextPage() {
	ff.curOffset += ff.limit
}

func (ff *FollowingFetcher) HasMorePage() bool {
	return ff.total == -1 || ff.curOffset < ff.total
}

func (ff *FollowingFetcher) GetNextPageFollowing() (*FollowingInfo, error) {
	folInfo, err := ff.client.GetFollowing(ff.uid, ff.curOffset, ff.limit)
	if err != nil {
		return nil, err
	}
	if folInfo.Total > 0 {
		ff.total = folInfo.Total
	}
	return folInfo, nil
}
