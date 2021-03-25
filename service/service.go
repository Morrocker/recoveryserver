package service

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morrocker/errors"
	"github.com/morrocker/log"
	"github.com/morrocker/recoveryserver/director"
)

// Service contains all the information used to run a successful service.
type Service struct {
	Director director.Director
	listener net.Listener

	mu sync.Mutex
	s  *http.Server
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by RunTLS so dead TCP connections
// (e.g. closing laptop mid-download) eventually go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// New returns an instance of a service.
func New(addr string) (*Service, error) {
	log.Info("Serving Service on address %s", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Service{
		listener: tcpKeepAliveListener{ln.(*net.TCPListener)},
	}, nil
}

var setGinModeOnce sync.Once

// Handler returns an http.Handler of the service.
func (s *Service) Handler() http.Handler {
	setGinModeOnce.Do(func() {
		gin.SetMode(gin.ReleaseMode)
	})

	mux := gin.New()

	// mux.Use(gin.Recovery())
	mux.Use(s.handleCORS)
	// mux.Use(s.handleAuth)
	// mux.Use(s.monitorHandler())

	mux.POST("/add", s.addRecovery)
	mux.POST("/change_priority", s.changePriority)
	mux.POST("/set_output", s.setOutput)
	mux.GET("/precalculate", s.precalculateSize)
	mux.GET("/recoveries", s.getRecoveries)
	// Recoveries run manipulation
	mux.GET("/queue_recovery", s.queueRecovery)
	mux.GET("/start_recovery", s.startRecovery)
	mux.GET("/pause_recovery", s.pauseRecovery)
	mux.GET("/cancel_recovery", s.cancelRecovery)
	// PDF generation
	mux.GET("/generate_delivery", s.writeDelivery)
	// Disk operations
	mux.GET("/devices", s.getDevices)
	mux.GET("/mount", s.mountDevice)
	mux.GET("/unmount", s.unmountDevice)
	// Requests
	mux.GET("/recovery_size", s.getDevices) // TODO

	mux.GET("/shutdown", s.shutdown)

	// mux.GET("/test", s.test)

	return mux
}

func (s *Service) server() *http.Server {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.s == nil {
		s.s = &http.Server{
			Handler: s.Handler(),
			Addr:    s.listener.Addr().String(),
		}
	}
	return s.s
}

// Close immediately closes the undelying Service's Server.
//
// Close returns any error returned from closing the Service's
// underlying Server.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.s == nil {
		return nil
	}

	return s.s.Close()
}

// Serve accepts incoming connections Service's listener, creating a
// new service goroutine for each. The service goroutines read requests and
// then call s.Handler() to reply to them.
func (s *Service) Serve() error {
	return s.server().Serve(s.listener)
}

// ServeTLS accepts incoming connections Service's listener, creating a
// new service goroutine for each. The service goroutines read requests and
// then call s.Handler() to reply to them.
//
// Additionally, files containing a certificate and matching private key for
// the Service must be provided. If the certificate is signed by
// a certificate authority, the certFile should be the concatenation of the
// Service's certificate, any intermediates, and the CA's certificate.
func (s *Service) ServeTLS(certFile, keyFile string) error {
	return s.server().ServeTLS(s.listener, certFile, keyFile)
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// StartDirector starts the recovery Director processes
func (s *Service) StartDirector() error {
	if err := s.Director.StartDirector(); err != nil {
		return errors.Extend("service.StartDirector()", err)
	}
	return nil
}
