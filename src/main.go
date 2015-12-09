package main

import (
	"servant/conf"
	"servant/server"
	"fmt"
)

func main() {
	xconf, err := conf.XConfigFromFile("conf/example.xml")
	if err != nil {
		fmt.Printf("read config error: %s", err)
		return
	}
	server.NewServer(xconf.ToConfig()).Run()
}
