package main

import (
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/server"
	"github.com/spf13/viper"
)

func init() {
	logger.ToggleTimestamp()
	config.SetFlags()
	logger.SetModes(viper.GetBool("verbose"), viper.GetBool("debug"), viper.GetBool("benchmar"))
	config.Data.Load()
}

func main() {
	server := server.New()
	server.StartServer()
	return
}
