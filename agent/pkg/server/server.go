package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/yusing/go-proxy/agent/pkg/env"
	"github.com/yusing/go-proxy/agent/pkg/handler"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/net/gphttp/server"
	"github.com/yusing/go-proxy/internal/task"
)

type Options struct {
	CACert, ServerCert *tls.Certificate
	Port               int
}

func StartAgentServer(parent task.Parent, opt Options) {
	t := parent.Subtask("agent_server")

	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: opt.CACert.Certificate[0]})
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertPEM)

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*opt.ServerCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	if env.AgentSkipClientCertCheck {
		tlsConfig.ClientAuth = tls.NoClientCert
	}

	logger := logging.GetLogger()
	agentServer := &http.Server{
		Handler:   handler.NewAgentHandler(),
		TLSConfig: tlsConfig,
	}

	go func() {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", opt.Port))
		if err != nil {
			server.HandleError(logger, err, "failed to listen on port")
			return
		}
		defer l.Close()
		if err := agentServer.Serve(tls.NewListener(l, tlsConfig)); err != nil {
			server.HandleError(logger, err, "failed to serve agent server")
		}
	}()

	logging.Info().Int("port", opt.Port).Msg("agent server started")

	go func() {
		defer t.Finish(nil)
		<-parent.Context().Done()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := agentServer.Shutdown(ctx)
		if err != nil {
			server.HandleError(logger, err, "failed to shutdown agent server")
		} else {
			logging.Info().Int("port", opt.Port).Msg("agent server stopped")
		}
	}()
}
