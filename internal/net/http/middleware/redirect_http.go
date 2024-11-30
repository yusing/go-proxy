package middleware

import (
	"net/http"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
)

var RedirectHTTP = &Middleware{
	before: func(next http.HandlerFunc, w ResponseWriter, r *Request) {
		if r.TLS == nil {
			r.URL.Scheme = "https"
			host := r.Host
			if i := strings.Index(host, ":"); i != -1 {
				host = host[:i] // strip port number if present
			}
			r.URL.Host = host + ":" + common.ProxyHTTPSPort
			logger.Info().Str("url", r.URL.String()).Msg("redirect to https")
			http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
			return
		}
		next(w, r)
	},
}
