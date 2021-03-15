package main

import (
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/server"
	"github.com/spf13/viper"
)

func init() {
	log.ToggleTimestamp()
	config.SetFlags()
	if viper.GetBool("debug") {
		log.SetMode("verbose")
	} else if viper.GetBool("verbose") {
		log.SetMode("debug")
	}
	config.Data.Load()
}

func main() {
	server := server.New()
	server.StartServer()
	return
}
