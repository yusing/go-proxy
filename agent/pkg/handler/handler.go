package handler

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"

	"github.com/yusing/go-proxy/agent/pkg/agent"
	"github.com/yusing/go-proxy/agent/pkg/certs"
	"github.com/yusing/go-proxy/agent/pkg/env"
	v1 "github.com/yusing/go-proxy/internal/api/v1"
	"github.com/yusing/go-proxy/internal/api/v1/utils"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/logging/memlogger"
	"github.com/yusing/go-proxy/internal/metrics/systeminfo"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type ServeMux struct{ *http.ServeMux }

func (mux ServeMux) HandleMethods(methods, endpoint string, handler http.HandlerFunc) {
	for _, m := range strutils.CommaSeperatedList(methods) {
		mux.ServeMux.HandleFunc(m+" "+agent.APIEndpointBase+endpoint, handler)
	}
}

func (mux ServeMux) HandleFunc(endpoint string, handler http.HandlerFunc) {
	mux.ServeMux.HandleFunc(agent.APIEndpointBase+endpoint, handler)
}

type NopWriteCloser struct {
	io.Writer
}

func (NopWriteCloser) Close() error {
	return nil
}

func NewAgentHandler() http.Handler {
	mux := ServeMux{http.NewServeMux()}

	mux.HandleFunc(agent.EndpointProxyHTTP+"/{path...}", ProxyHTTP)
	mux.HandleMethods("GET", agent.EndpointVersion, v1.GetVersion)
	mux.HandleMethods("GET", agent.EndpointName, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, env.AgentName)
	})
	mux.HandleMethods("GET", agent.EndpointHealth, CheckHealth)
	mux.HandleMethods("GET", agent.EndpointLogs, memlogger.HandlerFunc())
	mux.HandleMethods("GET", agent.EndpointSystemInfo, systeminfo.Poller.ServeHTTP)
	mux.ServeMux.HandleFunc("/", DockerSocketHandler())
	return mux
}

// NewRegistrationHandler creates a new registration handler
// It checks if the request is coming from an allowed host
// Generates a new client certificate and zips it
// Sends the zipped certificate to the client
// its run only once on agent first start.
func NewRegistrationHandler(task *task.Task, ca *tls.Certificate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !env.IsAllowedHost(r.RemoteAddr) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/done" {
			logging.Info().Msg("registration done")
			task.Finish(nil)
			w.WriteHeader(http.StatusOK)
			return
		}

		logging.Info().Msgf("received registration request from %s", r.RemoteAddr)

		caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Certificate[0]})

		crt, key, err := certs.NewClientCert(ca)
		if err != nil {
			utils.HandleErr(w, r, E.Wrap(err, "failed to generate client certificate"))
			return
		}

		zipped, err := certs.ZipCert(caPEM, crt, key)
		if err != nil {
			utils.HandleErr(w, r, E.Wrap(err, "failed to zip certificate"))
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		if _, err := w.Write(zipped); err != nil {
			logging.Error().Err(err).Msg("failed to respond to registration request")
			return
		}
	}
}
