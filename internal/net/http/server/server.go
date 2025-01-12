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
	"github.com/yusing/go-proxy/internal/common"
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

func StartServer(parent task.Parent, opt Options) (s *Server) {
	s = NewServer(opt)
	s.Start(parent)
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

	out := io.Discard
	if common.IsDebug {
		out = logging.GetLogger()
	}

	if opt.HTTPAddr != "" {
		httpSer = &http.Server{
			Addr:     opt.HTTPAddr,
			Handler:  opt.Handler,
			ErrorLog: log.New(out, "", 0), // most are tls related
		}
	}
	if certAvailable && opt.HTTPSAddr != "" {
		httpsSer = &http.Server{
			Addr:     opt.HTTPSAddr,
			Handler:  opt.Handler,
			ErrorLog: log.New(out, "", 0), // most are tls related
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
func (s *Server) Start(parent task.Parent) {
	if s.http == nil && s.https == nil {
		return
	}

	task := parent.Subtask("server."+s.Name, false)

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

	task.OnCancel("stop", s.stop)
}

func (s *Server) stop() {
	if s.http == nil && s.https == nil {
		return
	}

	ctx, cancel := context.WithTimeout(task.RootContext(), 3*time.Second)
	defer cancel()

	if s.http != nil && s.httpStarted {
		s.handleErr("http", s.http.Shutdown(ctx))
		s.httpStarted = false
		s.l.Info().Str("addr", s.http.Addr).Msgf("server stopped")
	}

	if s.https != nil && s.httpsStarted {
		s.handleErr("https", s.https.Shutdown(ctx))
		s.httpsStarted = false
		s.l.Info().Str("addr", s.https.Addr).Msgf("server stopped")
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
