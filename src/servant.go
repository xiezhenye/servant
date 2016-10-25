package main

import (
	"servant/conf"
	"servant/server"
	"fmt"
	"flag"
	"os"
	"github.com/davecgh/go-spew/spew"
)

type arrayFlags []string

func (self *arrayFlags) String() string {
	return fmt.Sprintf("%v", *self)
}

func (self *arrayFlags) Set(value string) error {
	*self = append(*self, value)
	return nil
}

func main() {
	var configs arrayFlags
	var configDirs arrayFlags
	var vars arrayFlags
	flag.Var(&configs, "conf", "config files path")
	flag.Var(&configDirs, "confdir", "config directories path")
	flag.Var(&vars, "var", "vars")
	var debug bool
	flag.BoolVar(&debug, "debug", false, "enable debug mode")
	flag.Parse()

	server.SetArgVars(vars)
	server.SetEnvVars()

	config, err := conf.LoadXmlConfig(configs, configDirs, server.CloneGlobalParams())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	if debug {
		config.Debug = true
		spew.Config.Indent = "    "
		spew.Config.MaxDepth = 100
		spew.Fdump(os.Stderr, config)
		//spew.Fprintf(os.Stderr, "%#v", config)
	}
	err = server.NewServer(&config).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}

