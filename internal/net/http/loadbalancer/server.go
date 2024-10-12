package loadbalancer

import (
	"net/http"

	"github.com/yusing/go-proxy/internal/net/types"
	U "github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	Server struct {
		_ U.NoCopy

		Name   string
		URL    types.URL
		Weight weightType

		handler   http.Handler
		healthMon health.HealthMonitor
	}
	servers []*Server
)

func NewServer(name string, url types.URL, weight weightType, handler http.Handler, healthMon health.HealthMonitor) *Server {
	srv := &Server{
		Name:      name,
		URL:       url,
		Weight:    weight,
		handler:   handler,
		healthMon: healthMon,
	}
	return srv
}

func (srv *Server) String() string {
	return srv.Name
}

func (srv *Server) IsHealthy() bool {
	return srv.healthMon.IsHealthy()
}
