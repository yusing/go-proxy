package http

import (
	"net/http"
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

func FilterHeaders(h http.Header, allowed []string) http.Header {
	if len(allowed) == 0 {
		return h
	}

	filtered := make(http.Header)

	for i, header := range allowed {
		values := h.Values(header)
		if len(values) == 0 {
			continue
		}
		filtered[http.CanonicalHeaderKey(allowed[i])] = append([]string(nil), values...)
	}

	return filtered
}

func HeaderToMap(h http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0] // Take the first value
		}
	}
	return result
}
