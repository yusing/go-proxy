package server

import (
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/autocert"
	"github.com/yusing/go-proxy/internal/common"
	"golang.org/x/net/context"
)

type Server struct {
	Name         string
	CertProvider *autocert.Provider
	http         *http.Server
	https        *http.Server
	httpStarted  bool
	httpsStarted bool
	startTime    time.Time
	task         common.Task
}

type Options struct {
	Name            string
	HTTPAddr        string
	HTTPSAddr       string
	CertProvider    *autocert.Provider
	RedirectToHTTPS bool
	Handler         http.Handler
}

type LogrusWrapper struct {
	*logrus.Entry
}

func (l LogrusWrapper) Write(b []byte) (int, error) {
	return l.Logger.WriterLevel(logrus.ErrorLevel).Write(b)
}

func NewServer(opt Options) (s *Server) {
	var httpSer, httpsSer *http.Server
	var httpHandler http.Handler

	logger := log.Default()
	logger.SetOutput(LogrusWrapper{
		logrus.WithFields(logrus.Fields{"?": "server", "name": opt.Name}),
	})

	certAvailable := false
	if opt.CertProvider != nil {
		_, err := opt.CertProvider.GetCert(nil)
		certAvailable = err == nil
	}

	if certAvailable && opt.RedirectToHTTPS && opt.HTTPSAddr != "" {
		httpHandler = redirectToTLSHandler(opt.HTTPSAddr)
	} else {
		httpHandler = opt.Handler
	}

	if opt.HTTPAddr != "" {
		httpSer = &http.Server{
			Addr:     opt.HTTPAddr,
			Handler:  httpHandler,
			ErrorLog: logger,
		}
	}
	if certAvailable && opt.HTTPSAddr != "" {
		httpsSer = &http.Server{
			Addr:     opt.HTTPSAddr,
			Handler:  opt.Handler,
			ErrorLog: logger,
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
		task:         common.GlobalTask("Server " + opt.Name),
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
		s.httpStarted = true
		logrus.Printf("starting http %s server on %s", s.Name, s.http.Addr)
		go func() {
			s.handleErr("http", s.http.ListenAndServe())
		}()
	}

	if s.https != nil {
		s.httpsStarted = true
		logrus.Printf("starting https %s server on %s", s.Name, s.https.Addr)
		go func() {
			s.handleErr("https", s.https.ListenAndServeTLS(s.CertProvider.GetCertPath(), s.CertProvider.GetKeyPath()))
		}()
	}

	go func() {
		<-s.task.Context().Done()
		s.stop()
		s.task.Finished()
	}()
}

func (s *Server) stop() {
	if s.http == nil && s.https == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if s.http != nil && s.httpStarted {
		s.handleErr("http", s.http.Shutdown(ctx))
		s.httpStarted = false
		logger.Debugf("HTTP server %q stopped", s.Name)
	}

	if s.https != nil && s.httpsStarted {
		s.handleErr("https", s.https.Shutdown(ctx))
		s.httpsStarted = false
		logger.Debugf("HTTPS server %q stopped", s.Name)
	}
}

func (s *Server) Uptime() time.Duration {
	return time.Since(s.startTime)
}

func (s *Server) handleErr(scheme string, err error) {
	switch {
	case err == nil, errors.Is(err, http.ErrServerClosed):
		return
	default:
		logrus.Fatalf("failed to start %s %s server: %s", scheme, s.Name, err)
	}
}

func redirectToTLSHandler(port string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Scheme = "https"
		r.URL.Host = r.URL.Hostname() + port

		var redirectCode int
		if r.Method == http.MethodGet {
			redirectCode = http.StatusMovedPermanently
		} else {
			redirectCode = http.StatusPermanentRedirect
		}
		http.Redirect(w, r, r.URL.String(), redirectCode)
	}
}

var logger = logrus.WithField("module", "server")
