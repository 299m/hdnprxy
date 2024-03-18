package main

import (
	"flag"
	"hdnprxy/service"
)

func main() {

	cfgpath := ""
	flag.StringVar(&cfgpath, "config", "", "Path to the configuration file(s)")
	flag.Parse()

	///Start the service
	service.ListenAndServeTls(cfgpath)
}
