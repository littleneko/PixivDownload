package pkg

import (
	"encoding/json"
	"strconv"
)

type IdWrapper string

func (w *IdWrapper) UnmarshalJSON(data []byte) (err error) {
	if zip, err := strconv.Atoi(string(data)); err == nil {
		str := strconv.Itoa(zip)
		*w = IdWrapper(str)
		return nil
	}
	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), w)
}

type Resp struct {
	Error   bool            `json:"error"`
	Message string          `json:"message"`
	Body    json.RawMessage `json:"body"`
}

type BookmarkWorks struct {
	Id        IdWrapper `json:"id"`
	Title     string    `json:"title"`
	Url       string    `json:"url"`
	UserId    IdWrapper `json:"userId"`
	UserName  string    `json:"userName"`
	PageCount int       `json:"pageCount"`
}

type BookmarkBody struct {
	Total int             `json:"total"`
	Works []BookmarkWorks `json:"works"`
}

type Urls struct {
	//Mini     string `json:"mini"`
	//Thumb    string `json:"thumb"`
	//Small    string `json:"small"`
	//Regular  string `json:"regular"`
	Original string `json:"original"`
}

type Illust struct {
	Id          IdWrapper `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Urls        Urls      `json:"urls"`
	//Tags        json.RawMessage `json:"tags"`
	UserId      IdWrapper `json:"userId"`
	UserName    string    `json:"userName"`
	UserAccount string    `json:"userAccount"`
	CreatedDate string    `json:"createdDate"`
	PageCount   int       `json:"pageCount"`
	CurPage     int       `json:"curPage"`
}
