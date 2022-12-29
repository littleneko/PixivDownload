package main

import "flag"

var configFileName = flag.String("config", "pixiv.toml", "config file name")

func main() {
	flag.Parse()
	conf := GetConfig(*configFileName)
	InitLog(conf)
	db := GetDB(conf)
	Start(conf, db)
}
