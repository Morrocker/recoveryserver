package main

import (
	log "github.com/Morrocker/logger"
	"github.com/recoveryserver/config"
	"github.com/spf13/viper"
)

const addr string = "localhost:5000"

func init() {
	log.ToggleTimestamp()
}

func main() {

	config.SetFlags()
	configName := viper.GetString("config")
	config.LoadConfig(configName)

	// log.Info("%s", viper.Get("config"))
	// log.Info("%v", viper.GetBool("debug"))

	// cancel := make(chan struct{})
	// errc := make(chan error)
	// errc2 := make(chan error)
	// logger.Info("Creating service with address %s", addr)
	// s, err := service.New(addr)
	// if err != nil {
	// 	logger.Error("%v", err)
	// }

	// go func() {
	// 	logger.Info("Starting server on %s", addr)
	// 	errc <- s.Serve()
	// }()

	// defer close(cancel)
	// select {
	// case err := <-errc:
	// 	logger.Error("Server error: %v", err)
	// 	errc2 <- err
	// }

	// return
}
