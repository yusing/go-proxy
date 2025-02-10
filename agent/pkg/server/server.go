package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/yusing/go-proxy/agent/pkg/handler"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
)

type Options struct {
	CACert, ServerCert *tls.Certificate
	Port               int
}

func StartAgentServer(parent task.Parent, opt Options) {
	t := parent.Subtask("agent server")

	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: opt.CACert.Certificate[0]})
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertPEM)

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*opt.ServerCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	if common.IsDebug {
		tlsConfig.ClientAuth = tls.NoClientCert
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", opt.Port))
	if err != nil {
		logging.Fatal().Err(err).Int("port", opt.Port).Msg("failed to listen on port")
		return
	}

	server := &http.Server{
		Handler:   handler.NewHandler(),
		TLSConfig: tlsConfig,
		ErrorLog:  log.New(logging.GetLogger(), "", 0),
	}
	go func() {
		defer l.Close()
		if err := server.Serve(tls.NewListener(l, tlsConfig)); err != nil {
			logging.Fatal().Err(err).Int("port", opt.Port).Msg("failed to serve")
		}
	}()

	logging.Info().Int("port", opt.Port).Msg("agent server started")

	go func() {
		defer t.Finish(nil)
		<-parent.Context().Done()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			logging.Error().Err(err).Int("port", opt.Port).Msg("failed to shutdown agent server")
		}
		logging.Info().Int("port", opt.Port).Msg("agent server stopped")
	}()
}
