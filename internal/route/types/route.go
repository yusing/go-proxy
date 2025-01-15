package types

import (
	"net/http"

	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	HTTPRoute interface {
		Entry
		http.Handler
		Health() health.Status
	}
	StreamRoute interface {
		Entry
		net.Stream
		Health() health.Status
	}
)
