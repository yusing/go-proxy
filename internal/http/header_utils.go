package http

import (
	"net/http"
	"slices"
)

func RemoveHop(h http.Header) {
	reqUpType := UpgradeType(h)
	RemoveHopByHopHeaders(h)

	if reqUpType != "" {
		h.Set("Connection", "Upgrade")
		h.Set("Upgrade", reqUpType)
	} else {
		h.Del("Connection")
	}
}

func CopyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func FilterHeaders(h http.Header, allowed []string) {
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
