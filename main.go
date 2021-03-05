package main

import (
	"github.com/morrocker/logger"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/server"
	"github.com/spf13/viper"
)

func init() {
	// logger.ToggleTimestamp()
	logger.Info("Setting flags")
	config.SetFlags()
	logger.Info("Loading server config")
	config.LoadConfig()
	logger.SetModes(viper.GetBool("verbose"), viper.GetBool("debug"))
}

func main() {
	logger.Info("Starting Recovery Server")
	server := server.New()
	server.StartServer()
	return
}
