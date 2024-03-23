package main

import (
	"crypto/tls"
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

func NewServer(name string, provider AutoCertProvider, httpAddr string, httpHandler http.Handler, httpsAddr string, httpsHandler http.Handler) *Server {
	if provider != nil {
		return &Server{
			Name:         name,
			CertProvider: provider,
			http: &http.Server{
				Addr:    httpAddr,
				Handler: httpHandler,
			},
			https: &http.Server{
				Addr:    httpsAddr,
				Handler: httpsHandler,
				TLSConfig: &tls.Config{
					GetCertificate: provider.GetCert,
				},
			},
		}
	}
	return &Server{
		Name:     name,
		KeyFile:  keyFileDefault,
		CertFile: certFileDefault,
		http: &http.Server{
			Addr:    httpAddr,
			Handler: httpHandler,
		},
		https: &http.Server{
			Addr:    httpsAddr,
			Handler: httpsHandler,
		},
	}
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

	if s.https != nil && (s.CertProvider != nil || utils.fileOK(s.CertFile) && utils.fileOK(s.KeyFile)) {
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
