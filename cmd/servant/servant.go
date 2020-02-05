package main

import (
	"flag"
	"fmt"
	"github.com/xiezhenye/servant/pkg/conf"
	"github.com/xiezhenye/servant/pkg/server"
	"os"
)

type arrayFlags []string

func (f *arrayFlags) String() string {
	return fmt.Sprintf("%v", *f)
}

func (f *arrayFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	var configs arrayFlags
	var configDirs arrayFlags
	var vars arrayFlags
	showVer := false
	flag.Var(&configs, "conf", "config files path")
	flag.Var(&configDirs, "confdir", "config directories path")
	flag.Var(&vars, "var", "vars")
	flag.BoolVar(&showVer, "ver", false, "show version and exit")
	//var debug bool
	//flag.BoolVar(&debug, "debug", false, "enable debug mode")
	flag.Parse()

	if showVer {
		fmt.Printf("%s-%s @%s\n", conf.Version, conf.Release, conf.Rev)
		return
	}

	server.SetArgVars(vars)
	server.SetEnvVars()

	config, err := conf.LoadXmlConfig(configs, configDirs, server.CloneGlobalParams())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	/*
		if debug {
			config.Debug = true
			spew.Config.Indent = "    "
			spew.Config.MaxDepth = 100
			spew.Fdump(os.Stderr, config)
		}*/
	err = server.NewServer(&config).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}
