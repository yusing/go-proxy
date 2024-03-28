package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type Server struct {
	Name         string
	KeyFile      string
	CertFile     string
	CertProvider AutoCertProvider
	http         *http.Server
	https        *http.Server
	httpStarted  bool
	httpsStarted bool
}

type ServerOptions struct {
	Name            string
	HTTPAddr        string
	HTTPSAddr       string
	CertProvider    AutoCertProvider
	RedirectToHTTPS bool
	Handler         http.Handler
}

type LogrusWrapper struct {
	l *logrus.Entry
}

func (l LogrusWrapper) Write(b []byte) (int, error) {
	return l.l.Logger.WriterLevel(logrus.ErrorLevel).Write(b)
}

func NewServer(opt ServerOptions) *Server {
	var httpHandler http.Handler
	var s *Server
	if opt.RedirectToHTTPS {
		httpHandler = http.HandlerFunc(redirectToTLSHandler)
	} else {
		httpHandler = opt.Handler
	}
	logger := log.Default()
	logger.SetOutput(LogrusWrapper{
		logrus.WithFields(logrus.Fields{"component": "server", "name": opt.Name}),
	})
	if opt.CertProvider != nil {
		s = &Server{
			Name:         opt.Name,
			CertProvider: opt.CertProvider,
			http: &http.Server{
				Addr:     opt.HTTPAddr,
				Handler:  httpHandler,
				ErrorLog: logger,
			},
			https: &http.Server{
				Addr:     opt.HTTPSAddr,
				Handler:  opt.Handler,
				ErrorLog: logger,
				TLSConfig: &tls.Config{
					GetCertificate: opt.CertProvider.GetCert,
				},
			},
		}
	}
	s = &Server{
		Name:     opt.Name,
		KeyFile:  keyFileDefault,
		CertFile: certFileDefault,
		http: &http.Server{
			Addr:     opt.HTTPAddr,
			Handler:  httpHandler,
			ErrorLog: logger,
		},
		https: &http.Server{
			Addr:     opt.HTTPSAddr,
			Handler:  opt.Handler,
			ErrorLog: logger,
		},
	}
	if !s.certsOK() {
		s.http.Handler = opt.Handler
	}
	return s
}

func (s *Server) Start() {
	if s.http != nil {
		s.httpStarted = true
		logrus.Printf("starting http %s server on %s", s.Name, s.http.Addr)
		go func() {
			err := s.http.ListenAndServe()
			s.handleErr("http", err)
		}()
	}

	if s.https != nil && (s.CertProvider != nil || s.certsOK()) {
		s.httpsStarted = true
		logrus.Printf("starting https %s server on %s", s.Name, s.https.Addr)
		go func() {
			err := s.https.ListenAndServeTLS(s.CertFile, s.KeyFile)
			s.handleErr("https", err)
		}()
	}
}

func (s *Server) Stop() {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

	if s.httpStarted {
		errHTTP := s.http.Shutdown(ctx)
		s.handleErr("http", errHTTP)
		s.httpStarted = false
	}

	if s.httpsStarted {
		errHTTPS := s.https.Shutdown(ctx)
		s.handleErr("https", errHTTPS)
		s.httpsStarted = false
	}
}

func (s *Server) handleErr(scheme string, err error) {
	switch err {
	case nil, http.ErrServerClosed:
		return
	default:
		logrus.Fatalf("failed to start %s %s server: %v", scheme, s.Name, err)
	}
}

func (s *Server) certsOK() bool {
	return utils.fileOK(s.CertFile) && utils.fileOK(s.KeyFile)
}