package middleware

import (
	"net/http"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/logging"
)

type redirectHTTP struct{}

var RedirectHTTP = NewMiddleware[redirectHTTP]()

// before implements RequestModifier.
func (redirectHTTP) before(w http.ResponseWriter, r *http.Request) (proceed bool) {
	if r.TLS != nil {
		return true
	}
	r.URL.Scheme = "https"
	host := r.Host
	if i := strings.Index(host, ":"); i != -1 {
		host = host[:i] // strip port number if present
	}
	r.URL.Host = host + ":" + common.ProxyHTTPSPort
	logging.Debug().Str("url", r.URL.String()).Msg("redirect to https")
	http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
	return true
}
