package server

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
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

	logger := logging.With().Str("server", opt.Name).Logger()

	certAvailable := false
	if opt.CertProvider != nil {
		_, err := opt.CertProvider.GetCert(nil)
		certAvailable = err == nil
	}

	out := io.Discard
	if common.IsDebug {
		out = logger
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
			err := s.http.ListenAndServe()
			if err != nil {
				s.handleErr(err, "failed to serve http server")
			}
		}()
		s.httpStarted = true
		s.l.Info().Str("addr", s.http.Addr).Msg("server started")
	}

	if s.https != nil {
		go func() {
			l, err := net.Listen("tcp", s.https.Addr)
			if err != nil {
				s.handleErr(err, "failed to listen on port")
				return
			}
			defer l.Close()
			s.handleErr(s.https.Serve(tls.NewListener(l, s.https.TLSConfig)), "failed to serve https server")
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

	ctx, cancel := context.WithTimeout(task.RootContext(), 5*time.Second)
	defer cancel()

	if s.http != nil && s.httpStarted {
		err := s.http.Shutdown(ctx)
		if err != nil {
			s.handleErr(err, "failed to shutdown http server")
		} else {
			s.httpStarted = false
			s.l.Info().Str("addr", s.http.Addr).Msgf("server stopped")
		}
	}

	if s.https != nil && s.httpsStarted {
		err := s.https.Shutdown(ctx)
		if err != nil {
			s.handleErr(err, "failed to shutdown https server")
		} else {
			s.httpsStarted = false
			s.l.Info().Str("addr", s.https.Addr).Msgf("server stopped")
		}
	}
}

func (s *Server) Uptime() time.Duration {
	return time.Since(s.startTime)
}

func (s *Server) handleErr(err error, msg string) {
	HandleError(&s.l, err, msg)
}
