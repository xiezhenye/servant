package main

import (
	"servant/conf"
	"servant/server"
	"fmt"
	"flag"
)

func main() {
	confPath := flag.String("conf", "conf/servant.xml", "config file path")
	flag.Parse()
	xconf, err := conf.XConfigFromFile(*confPath)
	if err != nil {
		fmt.Printf("read config error: %s\n", err)
		return
	}
	server.NewServer(xconf.ToConfig()).Run()
}
