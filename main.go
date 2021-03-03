package main

import (
	log "github.com/Morrocker/logger"
	"github.com/recoveryserver/server"
)

const addr string = "localhost:5000"

func init() {
	log.ToggleTimestamp()
}

func main() {
	server := server.New()

	server.StartService(addr)
	return
}
