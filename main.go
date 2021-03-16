package main

import (
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/server"
)

func init() {
	config.SetFlags()
	config.Data.Load()
	config.SetLogger()
}

func main() {
	server := server.New()
	log.Task("Starting Server")
	server.StartServer()
	return
}
