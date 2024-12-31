package server

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/autocert"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
)

type Server struct {
	Name         string
	CertProvider *autocert.Provider
	http         *http.Server
	https        *http.Server
	httpStarted  bool
	httpsStarted bool
	startTime    time.Time

	l zerolog.Logger
}

type Options struct {
	Name         string
	HTTPAddr     string
	HTTPSAddr    string
	CertProvider *autocert.Provider
	Handler      http.Handler
}

func StartServer(opt Options) (s *Server) {
	s = NewServer(opt)
	s.Start()
	return s
}

func NewServer(opt Options) (s *Server) {
	var httpSer, httpsSer *http.Server

	logger := logging.With().Str("module", "server").Str("name", opt.Name).Logger()

	certAvailable := false
	if opt.CertProvider != nil {
		_, err := opt.CertProvider.GetCert(nil)
		certAvailable = err == nil
	}

	if opt.HTTPAddr != "" {
		httpSer = &http.Server{
			Addr:     opt.HTTPAddr,
			Handler:  opt.Handler,
			ErrorLog: log.New(io.Discard, "", 0), // most are tls related
		}
	}
	if certAvailable && opt.HTTPSAddr != "" {
		httpsSer = &http.Server{
			Addr:     opt.HTTPSAddr,
			Handler:  opt.Handler,
			ErrorLog: log.New(io.Discard, "", 0), // most are tls related
			TLSConfig: &tls.Config{
				GetCertificate: opt.CertProvider.GetCert,
			},
		}
	}
	return &Server{
		Name:         opt.Name,
		CertProvider: opt.CertProvider,
		http:         httpSer,
		https:        httpsSer,
		l:            logger,
	}
}

// Start will start the http and https servers.
//
// If both are not set, this does nothing.
//
// Start() is non-blocking.
func (s *Server) Start() {
	if s.http == nil && s.https == nil {
		return
	}

	s.startTime = time.Now()
	if s.http != nil {
		go func() {
			s.handleErr("http", s.http.ListenAndServe())
		}()
		s.httpStarted = true
		s.l.Info().Str("addr", s.http.Addr).Msg("server started")
	}

	if s.https != nil {
		go func() {
			s.handleErr("https", s.https.ListenAndServeTLS(s.CertProvider.GetCertPath(), s.CertProvider.GetKeyPath()))
		}()
		s.httpsStarted = true
		s.l.Info().Str("addr", s.https.Addr).Msgf("server started")
	}

	task.OnProgramExit("server."+s.Name+".stop", s.stop)
}

func (s *Server) stop() {
	if s.http == nil && s.https == nil {
		return
	}

	if s.http != nil && s.httpStarted {
		s.handleErr("http", s.http.Shutdown(task.RootContext()))
		s.httpStarted = false
	}

	if s.https != nil && s.httpsStarted {
		s.handleErr("https", s.https.Shutdown(task.RootContext()))
		s.httpsStarted = false
	}
}

func (s *Server) Uptime() time.Duration {
	return time.Since(s.startTime)
}

func (s *Server) handleErr(scheme string, err error) {
	switch {
	case err == nil, errors.Is(err, http.ErrServerClosed), errors.Is(err, context.Canceled):
		return
	default:
		s.l.Fatal().Err(err).Str("scheme", scheme).Msg("server error")
	}
}
