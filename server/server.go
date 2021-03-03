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

type Server struct {
	config     config.Config
	cancel     chan struct{}
	service    *service.Service
	recoveries map[string]recovery.RecoveryGroup
	Lock       sync.Mutex
}

func New() (server *Server) {
	server.cancel = make(chan struct{})
	server.recoveries = make(map[string]recovery.RecoveryGroup)
	return
}

func (s *Server) StartService(addr string) {
	errc := make(chan error)
	errc2 := make(chan error)
	log.Info("Creating service with address %s", addr)
	srv, err := service.New(addr)
	if err != nil {
		log.Error("%v", err)
	}
	s.service = srv

	go func() {
		log.Info("Starting server on %s", addr)
		errc <- s.service.Serve()
	}()

	defer close(s.cancel)
	select {
	case err := <-errc:
		logger.Error("Server error: %v", err)
		errc2 <- err
	}
}

func (s *Server) LoadConfig() {
	config.SetFlags()
	configName := viper.GetString("config")
	if conf, e := config.LoadConfig(configName); e != nil {
		log.Error("%s", e)
		return
	} else {
		s.config = conf
	}
}
