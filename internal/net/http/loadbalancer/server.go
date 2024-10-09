package loadbalancer

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/yusing/go-proxy/internal/net/types"
)

type (
	Server struct {
		Name    string
		URL     types.URL
		Weight  weightType
		handler http.Handler

		pinger    *http.Client
		available atomic.Bool
	}
	servers []*Server
)

func NewServer(name string, url types.URL, weight weightType, handler http.Handler) *Server {
	srv := &Server{
		Name:    name,
		URL:     url,
		Weight:  weight,
		handler: handler,
		pinger:  &http.Client{Timeout: 3 * time.Second},
	}
	srv.available.Store(true)
	return srv
}

func (srv *Server) checkUpdateAvail(ctx context.Context) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodHead,
		srv.URL.String(),
		nil,
	)
	if err != nil {
		logger.Error("failed to create request: ", err)
		srv.available.Store(false)
	}

	resp, err := srv.pinger.Do(req)
	if err == nil && resp.StatusCode != http.StatusServiceUnavailable {
		if !srv.available.Swap(true) {
			logger.Infof("server %s is up", srv.Name)
		}
	} else if err != nil {
		if srv.available.Swap(false) {
			logger.Warnf("server %s is down: %s", srv.Name, err)
		}
	} else {
		if srv.available.Swap(false) {
			logger.Warnf("server %s is down: status %s", srv.Name, resp.Status)
		}
	}
}

func (srv *Server) String() string {
	return srv.Name
}
