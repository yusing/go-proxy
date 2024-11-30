package middleware

import (
	"net"
	"strings"

	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

var SetXForwarded = &Middleware{
	before: Rewrite(func(req *Request) {
		req.Header.Del(gphttp.HeaderXForwardedFor)
		clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			req.Header.Set(gphttp.HeaderXForwardedFor, clientIP)
		}
	}),
}

var HideXForwarded = &Middleware{
	before: Rewrite(func(req *Request) {
		for k := range req.Header {
			if strings.HasPrefix(k, "X-Forwarded-") {
				req.Header.Del(k)
			}
		}
	}),
}
