package app

import (
	"encoding/json"
	"fmt"
	"strconv"
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

// BasicIllustInfo is the illust info get from users bookmarks or users artworks
type BasicIllustInfo struct {
	Id        PixivID `json:"id"`
	Title     string  `json:"title"`
	UserId    PixivID `json:"userId"`
	UserName  string  `json:"userName"`
	PageCount int32   `json:"pageCount"`
}

func (bi *BasicIllustInfo) DigestString() string {
	return fmt.Sprintf("[id: %s, title: %s, uid: %s, uname: %s, pages: %d]", bi.Id, bi.Title, bi.UserId, bi.UserName, bi.PageCount)
}

type BookmarksBody struct {
	Total int32             `json:"total"`
	Works []BasicIllustInfo `json:"works"`
}

type Urls struct {
	//Mini     string `json:"mini"`
	//Thumb    string `json:"thumb"`
	//Small    string `json:"small"`
	//Regular  string `json:"regular"`
	Original string `json:"original"`
}

type IllustInfo struct {
	Id          PixivID  `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Urls        Urls     `json:"urls"`
	Tags        []string `json:"tags,omitempty"`
	R18         bool     `json:"r18,omitempty"`
	UserId      PixivID  `json:"userId"`
	UserName    string   `json:"userName"`
	UserAccount string   `json:"userAccount"`
	CreatedDate string   `json:"createdDate"`
	PageCount   int      `json:"pageCount"`
	CurPage     int      `json:"curPage"`
}

func (i *IllustInfo) DigestString() string {
	return fmt.Sprintf("[id: %s, title: %s, uid: %s, uname: %s, count: %d, R18: %v]", i.Id, i.Title, i.UserId, i.UserName, i.PageCount, i.R18)
}
