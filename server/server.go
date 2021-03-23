package server

import (
	"sync"

	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/config"
	"github.com/morrocker/recoveryserver/service"
)

// Server groups all server information
type Server struct {
	cancel  chan struct{}
	Service *service.Service
	Lock    sync.Mutex
}

// New initializes and returns a new recovery Server
func New() *Server {
	var server Server
	server.cancel = make(chan struct{})
	return &server
}

// StartServer starts the recovry server and listens for requests
func (s *Server) StartServer() {
	op := "server.StartServer()"
	errc := make(chan error)
	done := make(chan interface{})

	log.Task("Creating Service with address %s", config.Data.HostAddr)
	srvc, err := service.New(config.Data.HostAddr)
	if err != nil {
		log.Errorln(errors.Extend(op, err))
	}
	s.Service = srvc

	go func() {
		errc <- s.Service.Serve()
	}()
	go func() {
		errc <- s.Service.StartDirector()
	}()

	defer close(s.cancel)
	select {
	case err := <-errc:
		log.Errorln(err)
	case <-done:
		log.Task("Shutting down server")

	}
}
