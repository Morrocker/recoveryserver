package server

import (
	"sync"

	"github.com/Morrocker/logger"
	log "github.com/Morrocker/logger"
	"github.com/recoveryserver/config"
	"github.com/recoveryserver/recovery"
	"github.com/recoveryserver/service"
	"github.com/spf13/viper"
)

// Server groups all server information
type Server struct {
	config  config.Config
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

// StartService starts the recovry server and listens for requests
func (s *Server) StartService(addr string) {
	errc := make(chan error)
	errc2 := make(chan error)
	log.Info("Creating service with address %s", addr)
	srv, err := service.New(addr)
	if err != nil {
		log.Error("%v", err)
	}
	s.Service = srv
	s.Service.Director.Recoveries = make(map[string]*recovery.Recovery)

	go func() {
		log.Info("Starting server on %s", addr)
		errc <- s.Service.Serve()
	}()

	defer close(s.cancel)
	select {
	case err := <-errc:
		logger.Error("Server error: %v", err)
		errc2 <- err
	}
}

// LoadConfig loads the configuration file into the server
func (s *Server) LoadConfig() {
	config.SetFlags()
	configName := viper.GetString("config")
	conf, err := config.LoadConfig(configName)
	if err != nil {
		log.Error("%s", err)
		return
	}
	s.config = conf
}
