package middleware

import (
	"net/http"
	"slices"

	gpHTTP "github.com/yusing/go-proxy/http"
)

func removeHop(h Header) {
	reqUpType := gpHTTP.UpgradeType(h)
	gpHTTP.RemoveHopByHopHeaders(h)

	if reqUpType != "" {
		h.Set("Connection", "Upgrade")
		h.Set("Upgrade", reqUpType)
	} else {
		h.Del("Connection")
	}
}

func copyHeader(dst, src Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func filterHeaders(h Header, allowed []string) {
	if allowed == nil {
		return
	}

	for i := range allowed {
		allowed[i] = http.CanonicalHeaderKey(allowed[i])
	}

	for key := range h {
		if !slices.Contains(allowed, key) {
			h.Del(key)
		}
	}
}
