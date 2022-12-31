package main

import (
	"flag"
	"os"
	"os/signal"
	"pixiv/pkg"
	"syscall"
)

var configFileName = flag.String("config", "", "config file name")

func main() {
	flag.Parse()
	conf := pkg.GetConfig(*configFileName)
	pkg.InitLog(conf)
	illustInfoManager := pkg.NewIllustInfoManager(conf)
	pkg.Start(conf, illustInfoManager)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
