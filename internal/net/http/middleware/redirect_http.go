package middleware

import (
	"net/http"

	"github.com/yusing/go-proxy/internal/common"
)

var RedirectHTTP = &Middleware{
	before: func(next http.Handler, w ResponseWriter, r *Request) {
		if r.TLS == nil {
			r.URL.Scheme = "https"
			r.URL.Host = r.URL.Hostname() + ":" + common.ProxyHTTPSPort
			http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	},
}
