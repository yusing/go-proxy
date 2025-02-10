package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"

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
	defer l.Close()

	server := &http.Server{
		Handler:   handler.NewHandler(caCertPEM),
		TLSConfig: tlsConfig,
		ErrorLog:  log.New(logging.GetLogger(), "", 0),
	}
	server.Serve(tls.NewListener(l, tlsConfig))
}
