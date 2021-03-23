package main

import (
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/server"
)

func init() {
	config.Data.Load()
	config.SetLogger()
	config.CreatePDFDir()
}

func main() {
	server := server.New()
	log.Task("Starting Server")
	server.StartServer()
}
