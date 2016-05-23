package main

import (
	"servant/conf"
	"servant/server"
	"fmt"
	"flag"
	"os"
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
	var vars arrayFlags
	flag.Var(&configs, "conf", "config files path")
	flag.Var(&vars, "var", "vars")
	flag.Parse()

	server.SetArgVars(vars)
	server.SetEnvVars()

	config := conf.Config{}
	for _, confPath := range(configs) {
		xconf, err := conf.XConfigFromFile(confPath, server.CloneGlobalParams())
		if err != nil {
			fmt.Printf("read config file '%s' failed: %s\n", confPath, err)
			return
		}
		xconf.IntoConfig(&config)
	}

	err := server.NewServer(&config).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
}
