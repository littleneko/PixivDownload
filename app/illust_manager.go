package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	pixiv "github.com/littleneko/pixiv-api-go"
	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

type DatabaseType int

const (
	DatabaseTypeInvalid DatabaseType = -1
	DatabaseTypeNone                 = iota
	DatabaseTypeSqlite
	DatabaseTypeMysql
)

var databaseTypes = func() map[string]DatabaseType {
	return map[string]DatabaseType{
		"NONE":   DatabaseTypeNone,
		"SQLITE": DatabaseTypeSqlite,
		"MYSQL":  DatabaseTypeMysql,
	}
}

func GetDatabaseType(typeStr string) DatabaseType {
	typeStrLower := strings.ToUpper(typeStr)
	t, ok := databaseTypes()[typeStrLower]
	if !ok {
		return DatabaseTypeInvalid
	}
	return t
}

type IllustInfoManager interface {
	GetIllustCount(pid string) (int32, error)
	IsIllustExist(pid string) (bool, error)
	IsIllustPageExist(pid string, page int) (bool, error)
	SaveIllust(illust *pixiv.IllustInfo, hash string, filename string) error
	GetIllustInfo(pid string, page int) (*pixiv.IllustInfo, error)
	CheckDatabaseAndFile() error
}

const (
	createTableSQL = `
	CREATE TABLE IF NOT EXISTS illust (
    	pid VARCHAR(64) NOT NULL, 
    	page int NOT NULL DEFAULT 0,
    	title VARCHAR(255) NOT NULL DEFAULT '',
    	url VARCHAR(512) NOT NULL,
	    r18 int NOT NULL DEFAULT 0,
	    tags TEXT,
    	description TEXT,
	    width int NOT NULL DEFAULT 0,
	    height int NOT NULL DEFAULT 0,
    	page_count int NOT NULL DEFAULT 1,
	    bookmarks_count int NOT NULL DEFAULT 0,
	    like_count int NOT NULL DEFAULT 0,
	    comment_count int NOT NULL DEFAULT 0,
	    view_count int NOT NULL DEFAULT 0,
	    create_date DATETIME NOT NULL DEFAULT '1970-01-01',
	    upload_date DATETIME NOT NULL DEFAULT '1970-01-01',
    	user_id VARCHAR(64) NOT NULL DEFAULT '',
    	user_name VARCHAR(128) NOT NULL DEFAULT '',
    	user_account VARCHAR(64) NOT NULL DEFAULT '',
    	sha1 VARCHAR(256) NOT NULL,
    	filename VARCHAR(256) NOT NULL,
    	created_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    	updated_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    PRIMARY KEY(pid, page)
    )`

	illustCntSql        = "SELECT COUNT(1) FROM illust WHERE pid = ?"
	illustPageCntSql    = "SELECT COUNT(1) FROM illust WHERE pid = ? AND page = ?"
	getIllustPageCntSql = "SELECT MAX(page_count) FROM illust WHERE pid = ?"
	saveIllustSql       = "REPLACE INTO illust VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
	getIllustSql        = "SELECT * FROM illust WHERE pid = ? AND page = ?"
)

func GetIllustInfoManager(options *PixivDlOptions) (IllustInfoManager, error) {
	dbType := GetDatabaseType(options.DatabaseType)
	switch dbType {
	case DatabaseTypeNone:
		return NewDummyIllustInfoMgr(), nil
	case DatabaseTypeSqlite:
		return NewSqliteIllustInfoMgr(options), nil
	default:
		log.Errorf("Not supported database type '%s'", options.DatabaseType)
	}
	return nil, errors.New(fmt.Sprintf("Not supported database type '%s'", options.DatabaseType))
}

// DummyIllustInfoMgr do nothing
type DummyIllustInfoMgr struct {
}

func NewDummyIllustInfoMgr() *DummyIllustInfoMgr {
	return &DummyIllustInfoMgr{}
}

func (d *DummyIllustInfoMgr) GetIllustCount(string) (int32, error) {
	return 0, nil
}

func (d *DummyIllustInfoMgr) IsIllustExist(string) (bool, error) {
	return false, nil
}

func (d *DummyIllustInfoMgr) IsIllustPageExist(string, int) (bool, error) {
	return false, nil
}

func (d *DummyIllustInfoMgr) SaveIllust(*pixiv.IllustInfo, string, string) error {
	return nil
}

func (d *DummyIllustInfoMgr) GetIllustInfo(string, int) (*pixiv.IllustInfo, error) {
	return nil, errors.New("not found")
}

func (d *DummyIllustInfoMgr) CheckDatabaseAndFile() error {
	return nil
}

type SqliteIllustInfoMgr struct {
	db *sql.DB
}

func NewSqliteIllustInfoMgr(options *PixivDlOptions) *SqliteIllustInfoMgr {
	err := CheckAndMkdir(options.SqlitePath)
	if err != nil {
		log.Fatalf("Failed to create database dir, msg: %s", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(options.SqlitePath, "pixiv.db"))
	if err != nil {
		log.Fatalf("Failed to open illustMgr, msg: %s", err)
	}

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table, msg: %s", err)
	}

	return &SqliteIllustInfoMgr{db: db}
}

func (ps *SqliteIllustInfoMgr) GetIllustCount(id string) (int32, error) {
	rows, err := ps.db.Query(illustCntSql, id)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var count int32 = 0
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}

func (ps *SqliteIllustInfoMgr) getIllustPageCnt(pid string) (int32, error) {
	rows, err := ps.db.Query(getIllustPageCntSql, pid)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var count int32 = 0
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (ps *SqliteIllustInfoMgr) IsIllustExist(pid string) (bool, error) {
	rows, err := ps.db.Query(illustCntSql, pid)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var count int32 = 0
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return false, err
		}
	}
	if count == 0 {
		return false, nil
	}

	pageCount, err := ps.getIllustPageCnt(pid)
	if err != nil {
		return false, err
	}
	return count == pageCount, nil
}

func (ps *SqliteIllustInfoMgr) IsIllustPageExist(pid string, page int) (bool, error) {
	rows, err := ps.db.Query(illustPageCntSql, pid, page)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var count int32 = 0
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return false, err
		}
	}

	return count == 1, nil
}

func (ps *SqliteIllustInfoMgr) SaveIllust(illust *pixiv.IllustInfo, hash string, filename string) error {
	tags, _ := json.Marshal(illust.Tags)
	_, err := ps.db.Exec(saveIllustSql,
		illust.Id, illust.PageIdx, illust.Title, illust.Urls.Original, illust.R18, tags, illust.Description, illust.Width, illust.Height,
		illust.PageCount, illust.BookmarkCount, illust.LikeCount, illust.CommentCount, illust.ViewCount, illust.CreateDate, illust.UploadDate,
		illust.UserId, illust.UserName, illust.UserAccount, hash, filename)
	if err != nil {
		return err
	}
	return nil
}

func (ps *SqliteIllustInfoMgr) GetIllustInfo(id string, page int) (*pixiv.IllustInfo, error) {
	rows, err := ps.db.Query(getIllustSql, id, page)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var illust pixiv.IllustInfo
	var (
		hash     string
		filename string
		ctime    time.Time
		utime    time.Time
	)
	for rows.Next() {
		var tags string
		err := rows.Scan(&illust.Id, &illust.PageIdx, &illust.Title, &illust.Urls.Original, &illust.R18, &tags, &illust.Description, &illust.Width, &illust.Height,
			&illust.PageCount, &illust.BookmarkCount, &illust.LikeCount, &illust.CommentCount, &illust.ViewCount, &illust.CreateDate, &illust.UploadDate,
			&illust.UserId, illust.UserName, &illust.UserAccount, &hash, &filename, &ctime, &utime)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tags), &illust.Tags)
	}

	return &illust, nil
}

func (ps *SqliteIllustInfoMgr) CheckDatabaseAndFile() error {
	return nil
}
