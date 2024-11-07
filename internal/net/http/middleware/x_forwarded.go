package middleware

import (
	"net"
	"strings"
)

const (
	xForwardedFor    = "X-Forwarded-For"
	xForwardedMethod = "X-Forwarded-Method"
	xForwardedHost   = "X-Forwarded-Host"
	xForwardedProto  = "X-Forwarded-Proto"
	xForwardedURI    = "X-Forwarded-Uri"
	xForwardedPort   = "X-Forwarded-Port"
)

var SetXForwarded = &Middleware{
	before: Rewrite(func(req *Request) {
		delXForwarded(req)
		clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			req.Header.Set(xForwardedFor, clientIP)
		} else {
			req.Header.Set(xForwardedFor, req.RemoteAddr)
		}
		req.Header.Set(xForwardedHost, req.Host)
		if req.TLS == nil {
			req.Header.Set(xForwardedProto, "http")
		} else {
			req.Header.Set(xForwardedProto, "https")
		}
	}),
}

var HideXForwarded = &Middleware{
	before: Rewrite(delXForwarded),
}

func delXForwarded(req *Request) {
	req.Header.Del("Forwarded")
	toRemove := make([]string, 0)
	for k := range req.Header {
		if strings.HasPrefix(k, "X-Forwarded-") {
			toRemove = append(toRemove, k)
		}
	}
	for _, k := range toRemove {
		req.Header.Del(k)
	}
}
