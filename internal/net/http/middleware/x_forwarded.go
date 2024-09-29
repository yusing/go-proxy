package middleware

import (
	"net"
	"strings"
)

var AddXForwarded = &Middleware{
	rewrite: func(req *Request) {
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

var SetXForwarded = &Middleware{
	rewrite: func(req *Request) {
		clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			prior := req.Header["X-Forwarded-For"]
			if len(prior) > 0 {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
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
