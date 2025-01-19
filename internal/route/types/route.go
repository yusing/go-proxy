package types

import (
	"net/http"

	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	Route interface {
		Entry
		HealthMonitor() health.HealthMonitor
	}
	HTTPRoute interface {
		Route
		http.Handler
	}
	StreamRoute interface {
		Route
		net.Stream
	}
)
