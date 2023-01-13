package main

import (
	"database/sql"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type PixivDB interface {
	IllustExist(id string) (bool, error)
	SaveIllust(illust *Illust, hash string, fileName string) error
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
)

type PixivSqlite struct {
	db *sql.DB
}

func GetDB(conf *Config) PixivDB {
	if conf.DatabaseType != "sqlite" {
		log.Fatalf("Not supported database type %s", conf.DatabaseType)
	}

	if _, err := os.Stat(conf.SqlitePath); os.IsNotExist(err) {
		err = os.MkdirAll(conf.SqlitePath, 0755)
		if err != nil {
			log.Fatalf("Failed to create database, msg: %s", err)
		}
	}

	db, err := sql.Open("sqlite3", filepath.Join(conf.SqlitePath, "pixiv.db"))
	if err != nil {
		log.Fatalf("Failed to open db, msg: %s", err)
	}

	_, err = db.Exec(CreateTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table, msg: %s", err)
	}
	_, err = db.Exec(CreateIndexSQL)
	if err != nil {
		log.Fatalf("Failed to create index, msg: %s", err)
	}

	return &PixivSqlite{db: db}
}

func (ps *PixivSqlite) IllustExist(id string) (bool, error) {
	rows, err := ps.db.Query("SELECT COUNT(1) FROM illust WHERE illust_id = ?", id)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return false, err
		}
	}

	return count > 0, nil
}

func (ps *PixivSqlite) SaveIllust(illust *Illust, hash string, fileName string) error {
	pid := fmt.Sprintf("%s_p%d", illust.Id, illust.CurPage)
	_, err := ps.db.Exec("REPLACE INTO illust VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		pid, illust.Id, illust.Title, illust.Urls.Original, illust.CurPage, illust.PageCount, illust.Description,
		illust.UserId, illust.UserName, illust.UserAccount, hash, fileName)
	if err != nil {
		return err
	}
	return nil
}
