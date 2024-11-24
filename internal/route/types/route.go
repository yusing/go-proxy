package types

import (
	"net/http"

	net "github.com/yusing/go-proxy/internal/net/types"
)

type (
	HTTPRoute interface {
		Entry
		http.Handler
	}
	StreamRoute interface {
		Entry
		net.Stream
	}
)
