package types

import (
	"net/http"
	"time"

	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	"github.com/yusing/go-proxy/internal/net/types"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	Server struct {
		_ U.NoCopy

		Name   string
		URL    types.URL
		Weight Weight

		handler   http.Handler
		healthMon health.HealthMonitor
	}
	Servers = []*Server
	Pool    = F.Map[string, *Server]
)

var NewServerPool = F.NewMap[Pool]

func NewServer(name string, url types.URL, weight Weight, handler http.Handler, healthMon health.HealthMonitor) *Server {
	srv := &Server{
		Name:      name,
		URL:       url,
		Weight:    weight,
		handler:   handler,
		healthMon: healthMon,
	}
	return srv
}

func (srv *Server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	srv.handler.ServeHTTP(rw, r)
}

func (srv *Server) String() string {
	return srv.Name
}

func (srv *Server) Status() health.Status {
	return srv.healthMon.Status()
}

func (srv *Server) Uptime() time.Duration {
	return srv.healthMon.Uptime()
}

func (srv *Server) TryWake() error {
	waker, ok := srv.handler.(idlewatcher.Waker)
	if ok {
		if err := waker.Wake(); err != nil {
			return err
		}
	}
	return nil
}

func (srv *Server) HealthMonitor() health.HealthMonitor {
	return srv.healthMon
}
