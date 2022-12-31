package pkg

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

const (
	BookmarksUrl   = "https://www.pixiv.net/ajax/user/%s/illusts/bookmarks"
	IllustUrl      = "https://www.pixiv.net/ajax/illust/%s"
	IllustPagesUrl = "https://www.pixiv.net/ajax/illust/%s/pages"
)

const (
	BookmarksReferUrl      = "https://www.pixiv.net/users/%s/bookmarks/artworks"
	IllustReferUrl         = "https://www.pixiv.net/artworks/%s"
	IllustDownloadReferUrl = "https://www.pixiv.net"
)

var (
	ErrNotFound        = errors.New("NotFound")
	ErrFailedUnmarshal = errors.New("FailedUnmarshal")
)

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
	pc := PixivClient{
		client: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
		header: header,
	}
	return &pc
}

func (p *PixivClient) request(url string, refer string) (*http.Response, error) {
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

func (p *PixivClient) getPixiv(url string, refer string) (*PixivResponse, error) {
	resp, err := p.request(url, refer)
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

func (p *PixivClient) getBookmarksUrl(uid string, offset int32, limit int32) string {
	params := url.Values{}
	params.Set("tag", "")
	params.Set("offset", strconv.FormatInt(int64(offset), 10))
	params.Set("limit", strconv.FormatInt(int64(limit), 10))
	params.Set("rest", "show")

	bmUrl, _ := url.Parse(fmt.Sprintf(BookmarksUrl, uid))
	bmUrl.RawQuery = params.Encode()

	return bmUrl.String()
}

func (p *PixivClient) GetBookmarks(uid string, offset int32, limit int32) (*BookmarksBody, error) {
	bUrl := p.getBookmarksUrl(uid, offset, limit)
	refer := fmt.Sprintf(BookmarksReferUrl, uid)
	resp, err := p.getPixiv(bUrl, refer)
	if err != nil {
		return nil, err
	}

	var bmBody BookmarksBody
	err = json.Unmarshal(resp.Body, &bmBody)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}
	return &bmBody, nil
}

func (p *PixivClient) GetIllustInfo(illustId PixivIDType) ([]*Illust, error) {
	illust, err := p.getIllustBasicInfo(illustId)
	if err != nil {
		return nil, err
	}
	if illust.PageCount == 1 {
		return []*Illust{illust}, nil
	} else {
		return p.getIllustAllPages(illust)
	}
}

func (p *PixivClient) getIllustBasicInfo(illustId PixivIDType) (*Illust, error) {
	illustUrl := fmt.Sprintf(IllustUrl, illustId)
	refer := fmt.Sprintf(IllustReferUrl, illustId)
	iResp, err := p.getPixiv(illustUrl, refer)
	if err != nil {
		return nil, err
	}

	var illust Illust
	err = json.Unmarshal(iResp.Body, &illust)
	if err != nil {
		return nil, ErrFailedUnmarshal
	}
	return &illust, nil
}

func (p *PixivClient) getIllustAllPages(seed *Illust) ([]*Illust, error) {
	illustUrl := fmt.Sprintf(IllustPagesUrl, seed.Id)
	refer := fmt.Sprintf(IllustReferUrl, seed.Id)
	iResp, err := p.getPixiv(illustUrl, refer)
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

	var illusts []*Illust
	for idx := range illustPageBody {
		illust := *seed
		illust.CurPage = idx
		illust.Urls = illustPageBody[idx].Urls
		illusts = append(illusts, &illust)
	}
	return illusts, nil
}

func (p *PixivClient) DownloadIllust(url string) ([]byte, error) {
	resp, err := p.request(url, IllustDownloadReferUrl)
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
