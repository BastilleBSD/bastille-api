package main

import (
    "bastille-api/api"
    "flag"
)

func main() {

	debug := flag.Bool("debug", false, "Enable debug logging")
	Config := flag.String("config", "", "Config file location")
	Port := flag.String("port", "", "API server port")

	flag.Parse()

	api.InitLogger(*debug)

	api.Start(*Config, *Port)
}
