package middleware

import (
	"net"
)

var SetXForwarded = &Middleware{
	rewrite: func(req *Request) {
		req.Header.Del("Forwarded")
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Forwarded-Host")
		req.Header.Del("X-Forwarded-Proto")
		clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			req.Header.Set("X-Forwarded-For", clientIP)
		} else {
			req.Header.Del("X-Forwarded-For")
		}
		req.Header.Set("X-Forwarded-Host", req.Host)
		if req.TLS == nil {
			req.Header.Set("X-Forwarded-Proto", "http")
		} else {
			req.Header.Set("X-Forwarded-Proto", "https")
		}
	},
}

var HideXForwarded = &Middleware{
	rewrite: func(req *Request) {
		req.Header.Del("Forwarded")
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Forwarded-Host")
		req.Header.Del("X-Forwarded-Proto")
	},
}
