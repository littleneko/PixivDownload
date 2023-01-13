package main

import (
	"flag"
	"os"
	"os/signal"
	"pixiv/pkg"
	"syscall"
)

var configFileName = flag.String("config", "pixiv.toml", "config file name")

func main() {
	flag.Parse()
	conf := pkg.GetConfig(*configFileName)
	pkg.InitLog(conf)
	db := pkg.GetDB(conf)
	pkg.Start(conf, db)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
