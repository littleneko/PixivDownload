package pkg

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type DatabaseType int

const (
	UnknownDatabaseType DatabaseType = -1
	DatabaseTypeSqlite               = iota
	DatabaseTypeMysql
)

var databaseTypes = func() map[string]DatabaseType {
	return map[string]DatabaseType{
		"sqlite": DatabaseTypeSqlite,
		"mysql":  DatabaseTypeMysql,
	}
}

func ParseDatabaseType(typeStr string) DatabaseType {
	typeStrLower := strings.ToLower(typeStr)
	t, ok := databaseTypes()[typeStrLower]
	if !ok {
		return UnknownDatabaseType
	}
	return t
}

type IllustInfoManager interface {
	IllustCount(id string) (int32, error)
	CheckIllust(id string, count int32) (bool, error)
	SaveIllust(illust *Illust, hash string, fileName string) error
	CheckDatabaseAndFile() error
}

const (
	CreateTableSQL = `
	CREATE TABLE IF NOT EXISTS illust (
    	pid VARCHAR(64) PRIMARY KEY, 
    	illust_id VARCHAR(64) NOT NULL,
    	title VARCHAR(255) NOT NULL,
    	url VARCHAR(512) NOT NULL,
    	cur_page int NOT NULL DEFAULT 0,
    	all_page int NOT NULL DEFAULT 1,
    	description TEXT,
    	uid VARCHAR(64) NOT NULL,
    	user_name VARCHAR(128) NOT NULL,
    	user_account VARCHAR(64) NOT NULL,
    	sha1 VARCHAR(256) NOT NULL,
    	file_path VARCHAR(256) NOT NULL,
    	created_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    	update_time DATETIME DEFAULT CURRENT_TIMESTAMP
    )`

	CreateIndexSQL = `CREATE INDEX IF NOT EXISTS idx_iid ON illust(illust_id)`

	CheckIllustSql = "SELECT COUNT(1) FROM illust WHERE illust_id = ?"
	SaveIllustSql  = "REPLACE INTO illust VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
)

type SqliteIllustInfoMgr struct {
	db *sql.DB
}

func NewIllustInfoManager(conf *Config) IllustInfoManager {
	if ParseDatabaseType(conf.DatabaseType) != DatabaseTypeSqlite {
		log.Fatalf("Not supported database type '%s'", conf.DatabaseType)
	}
	return NewSqliteIllustInfoMgr(conf)
}

func NewSqliteIllustInfoMgr(conf *Config) *SqliteIllustInfoMgr {
	err := CheckAndMkdir(conf.SqlitePath)
	if err != nil {
		log.Fatalf("Failed to create database dir, msg: %s", err)
	}

	db, err := sql.Open("sqlite3", filepath.Join(conf.SqlitePath, "pixiv.db"))
	if err != nil {
		log.Fatalf("Failed to open illustMgr, msg: %s", err)
	}

	_, err = db.Exec(CreateTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table, msg: %s", err)
	}
	_, err = db.Exec(CreateIndexSQL)
	if err != nil {
		log.Fatalf("Failed to create index, msg: %s", err)
	}

	return &SqliteIllustInfoMgr{db: db}
}

func (ps *SqliteIllustInfoMgr) IllustCount(id string) (int32, error) {
	rows, err := ps.db.Query(CheckIllustSql, id)
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

func (ps *SqliteIllustInfoMgr) CheckIllust(id string, count int32) (bool, error) {
	cnt, err := ps.IllustCount(id)
	if err != nil {
		return false, err
	}
	return cnt == count, nil
}

func (ps *SqliteIllustInfoMgr) SaveIllust(illust *Illust, hash string, fileName string) error {
	pid := fmt.Sprintf("%s_p%d", illust.Id, illust.CurPage)
	_, err := ps.db.Exec(SaveIllustSql,
		pid, illust.Id, illust.Title, illust.Urls.Original, illust.CurPage, illust.PageCount, illust.Description,
		illust.UserId, illust.UserName, illust.UserAccount, hash, fileName)
	if err != nil {
		return err
	}
	return nil
}

func (ps *SqliteIllustInfoMgr) CheckDatabaseAndFile() error {
	return nil
}
