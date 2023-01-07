package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
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
	IllustCount(id string) (int32, error)
	CheckIllust(id string, expectedCnt int32) (bool, error)
	IllustExist(id string, page int) (bool, error)
	SaveIllust(illust *IllustInfo, hash string, filename string) error
	GetIllustInfo(id string, page int) (*IllustInfo, error)
	CheckDatabaseAndFile() error
}

const (
	createTableSQL = `
	CREATE TABLE IF NOT EXISTS illust (
    	pid VARCHAR(64) PRIMARY KEY, 
    	illust_id VARCHAR(64) NOT NULL,
    	title VARCHAR(255) NOT NULL,
    	url VARCHAR(512) NOT NULL,
    	cur_page int NOT NULL DEFAULT 0,
    	all_page int NOT NULL DEFAULT 1,
	    tags TEXT,
    	description TEXT,
    	uid VARCHAR(64) NOT NULL,
    	user_name VARCHAR(128) NOT NULL,
    	user_account VARCHAR(64) NOT NULL,
    	sha1 VARCHAR(256) NOT NULL,
    	file_path VARCHAR(256) NOT NULL,
    	created_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    	updated_time DATETIME DEFAULT CURRENT_TIMESTAMP
    )`

	createIndexSQL = `CREATE INDEX IF NOT EXISTS idx_iid ON illust(illust_id)`

	checkIllustSql = "SELECT COUNT(1) FROM illust WHERE illust_id = ?"
	illustExistSql = "SELECT COUNT(1) FROM illust WHERE pid = ?"
	saveIllustSql  = "REPLACE INTO illust VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
	getIllustSql   = "SELECT illust_id, title, url, cur_page, all_page, tags, description, uid, user_name, user_account FROM illust WHERE pid = ?"
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

func (d *DummyIllustInfoMgr) IllustCount(string) (int32, error) {
	return 0, nil
}

func (d *DummyIllustInfoMgr) CheckIllust(string, int32) (bool, error) {
	return false, nil
}

func (d *DummyIllustInfoMgr) IllustExist(id string, page int) (bool, error) {
	return false, nil
}

func (d *DummyIllustInfoMgr) SaveIllust(*IllustInfo, string, string) error {
	return nil
}

func (d *DummyIllustInfoMgr) GetIllustInfo(id string, page int) (*IllustInfo, error) {
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

	db, err := sql.Open("sqlite3", filepath.Join(options.SqlitePath, "pixiv.db"))
	if err != nil {
		log.Fatalf("Failed to open illustMgr, msg: %s", err)
	}

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table, msg: %s", err)
	}
	_, err = db.Exec(createIndexSQL)
	if err != nil {
		log.Fatalf("Failed to create index, msg: %s", err)
	}

	return &SqliteIllustInfoMgr{db: db}
}

func (ps *SqliteIllustInfoMgr) IllustCount(id string) (int32, error) {
	rows, err := ps.db.Query(checkIllustSql, id)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var count int32 = 0
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}

func (ps *SqliteIllustInfoMgr) CheckIllust(id string, expectedCnt int32) (bool, error) {
	cnt, err := ps.IllustCount(id)
	if err != nil {
		return false, err
	}
	return cnt == expectedCnt, nil
}

func (ps *SqliteIllustInfoMgr) IllustExist(id string, page int) (bool, error) {
	rows, err := ps.db.Query(illustExistSql, fmt.Sprintf("%s_%d", id, page))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var count int32 = 0
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return false, err
		}
	}

	return count == 1, nil
}

func (ps *SqliteIllustInfoMgr) SaveIllust(illust *IllustInfo, hash string, filename string) error {
	pid := fmt.Sprintf("%s_p%d", illust.Id, illust.CurPage)
	tags, _ := json.Marshal(illust.Tags)
	_, err := ps.db.Exec(saveIllustSql,
		pid, illust.Id, illust.Title, illust.Urls.Original, illust.CurPage, illust.PageCount, tags,
		illust.Description, illust.UserId, illust.UserName, illust.UserAccount, hash, filename)
	if err != nil {
		return err
	}
	return nil
}

func (ps *SqliteIllustInfoMgr) GetIllustInfo(id string, page int) (*IllustInfo, error) {
	rows, err := ps.db.Query(getIllustSql, fmt.Sprintf("%s_%d", id, page))
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var illust IllustInfo
	for rows.Next() {
		var tags string
		err := rows.Scan(&illust.Id, &illust.Title, &illust.Urls.Original, &illust.CurPage, &illust.PageCount, &tags,
			&illust.Description, &illust.UserId, illust.UserName, &illust.UserAccount)
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
