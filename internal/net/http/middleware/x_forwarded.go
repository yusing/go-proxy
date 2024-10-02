package middleware

import (
	"net"
	"net/http"
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
	before: func(next http.HandlerFunc, w ResponseWriter, req *Request) {
		req.Header.Del("Forwarded")
		req.Header.Del(xForwardedFor)
		req.Header.Del(xForwardedHost)
		req.Header.Del(xForwardedProto)
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
		next(w, req)
	},
}

var HideXForwarded = &Middleware{
	before: func(next http.HandlerFunc, w ResponseWriter, req *Request) {
		req.Header.Del("Forwarded")
		req.Header.Del(xForwardedFor)
		req.Header.Del(xForwardedHost)
		req.Header.Del(xForwardedProto)
		next(w, req)
	},
}
