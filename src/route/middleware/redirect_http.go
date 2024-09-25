package middleware

import (
	"net/http"

	"github.com/yusing/go-proxy/common"
)

var RedirectHTTP = &Middleware{
	before: func(w ResponseWriter, r *Request) (continue_ bool) {
		if r.TLS == nil {
			r.URL.Scheme = "https"
			r.URL.Host = r.URL.Hostname() + common.ProxyHTTPSPort
			http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
		} else {
			continue_ = true
		}
		return
	},
}
