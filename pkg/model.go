package pkg

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type PixivIDType string

func (w *PixivIDType) UnmarshalJSON(data []byte) (err error) {
	if zip, err := strconv.Atoi(string(data)); err == nil {
		str := strconv.Itoa(zip)
		*w = PixivIDType(str)
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

type BookmarkWork struct {
	Id        PixivIDType `json:"id"`
	Title     string      `json:"title"`
	Url       string      `json:"url"`
	UserId    PixivIDType `json:"userId"`
	UserName  string      `json:"userName"`
	PageCount int32       `json:"pageCount"`
}

type BookmarksBody struct {
	Total int32          `json:"total"`
	Works []BookmarkWork `json:"works"`
}

type Urls struct {
	//Mini     string `json:"mini"`
	//Thumb    string `json:"thumb"`
	//Small    string `json:"small"`
	//Regular  string `json:"regular"`
	Original string `json:"original"`
}

type Illust struct {
	Id          PixivIDType `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Urls        Urls        `json:"urls"`
	//Tags        json.RawMessage `json:"tags"`
	UserId      PixivIDType `json:"userId"`
	UserName    string      `json:"userName"`
	UserAccount string      `json:"userAccount"`
	CreatedDate string      `json:"createdDate"`
	PageCount   int         `json:"pageCount"`
	CurPage     int         `json:"curPage"`
}

func (i *Illust) DescriptionString() string {
	return fmt.Sprintf("id: %s, title: %s, uid: %s, uname: %s, count: %d", i.Id, i.Title, i.UserId, i.UserName, i.PageCount)
}
