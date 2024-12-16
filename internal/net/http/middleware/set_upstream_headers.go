package middleware

import (
	"net/http"

	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

// internal use only.
type setUpstreamHeaders struct {
	Scheme, Host, Port string
}

var suh = NewMiddleware[setUpstreamHeaders]()

func newSetUpstreamHeaders(rp *gphttp.ReverseProxy) *Middleware {
	m, err := suh.New(OptionsRaw{
		"scheme": rp.TargetURL.Scheme,
		"host":   rp.TargetURL.Hostname(),
		"port":   rp.TargetURL.Port(),
	})
	if err != nil {
		panic(err)
	}
	return m
}

// before implements RequestModifier.
func (s setUpstreamHeaders) before(w http.ResponseWriter, r *http.Request) (proceed bool) {
	r.Header.Set(gphttp.HeaderUpstreamScheme, s.Scheme)
	r.Header.Set(gphttp.HeaderUpstreamHost, s.Host)
	r.Header.Set(gphttp.HeaderUpstreamPort, s.Port)
	return true
}
