package server

import (
	"sync"

	"github.com/morrocker/logger"
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
	errc := make(chan error)
	errc2 := make(chan error)
	logger.InfoV("Creating service with address %s", config.Data.HostAddr)
	srv, err := service.New(config.Data.HostAddr)
	if err != nil {
		logger.Error("%v", err)
	}
	s.Service = srv

	go func() {
		logger.Info("Serving service on address %s", config.Data.HostAddr)
		errc <- s.Service.Serve()
	}()

	s.Service.StartDirector(config.Data)

	defer close(s.cancel)
	select {
	case err := <-errc:
		logger.Error("Server error: %v", err)
		errc2 <- err
	}
}
