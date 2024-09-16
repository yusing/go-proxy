package route

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/error"
	P "github.com/yusing/go-proxy/proxy"
	PT "github.com/yusing/go-proxy/proxy/fields"
	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	HTTPRoute struct {
		Alias PT.Alias `json:"alias"`

		TargetURL    URL
		PathPatterns PT.PathPatterns

		mux     *http.ServeMux
		handler *P.ReverseProxy
	}

	URL          url.URL
	PathKey      = PT.PathPattern
	SubdomainKey = PT.Alias
)

var httpRoutes = F.NewMap[SubdomainKey, *HTTPRoute]()

func NewHTTPRoute(entry *P.Entry) (*HTTPRoute, E.NestedError) {
	var tr *http.Transport
	if entry.NoTLSVerify {
		tr = transportNoTLS
	} else {
		tr = transport
	}

	rp := P.NewReverseProxy(entry.URL, tr, entry)

	httpRoutes.Lock()
	defer httpRoutes.Unlock()

	var r *HTTPRoute
	r, ok := httpRoutes.UnsafeGet(entry.Alias)
	if !ok {
		r = &HTTPRoute{
			Alias:        entry.Alias,
			TargetURL:    URL(*entry.URL),
			PathPatterns: entry.PathPatterns,
			handler:      rp,
		}
		httpRoutes.UnsafeSet(entry.Alias, r)
	}

	rewrite := rp.Rewrite

	if logrus.GetLevel() == logrus.DebugLevel {
		l := logrus.WithField("alias", entry.Alias)

		rp.Rewrite = func(pr *P.ProxyRequest) {
			l.Debug("request URL: ", pr.In.Host, pr.In.URL.Path)
			l.Debug("request headers: ", pr.In.Header)
			rewrite(pr)
		}
	} else {
		rp.Rewrite = rewrite
	}

	return r, E.Nil()
}

func (r *HTTPRoute) String() string {
	return string(r.Alias)
}

func (r *HTTPRoute) Start() E.NestedError {
	r.mux = http.NewServeMux()
	for _, p := range r.PathPatterns {
		r.mux.HandleFunc(string(p), r.handler.ServeHTTP)
	}
	httpRoutes.Set(r.Alias, r)
	return E.Nil()
}

func (r *HTTPRoute) Stop() E.NestedError {
	r.mux = nil
	httpRoutes.Delete(r.Alias)
	return E.Nil()
}

func (u *URL) String() string {
	return (*url.URL)(u).String()
}

func (u *URL) MarshalText() (text []byte, err error) {
	return []byte(u.String()), nil
}

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	mux, err := findMux(r.Host, PathKey(r.URL.Path))
	if err != nil {
		err = E.Failure("request").
			Subjectf("%s %s%s", r.Method, r.Host, r.URL.Path).
			With(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		logrus.Error(err)
		return
	}
	mux.ServeHTTP(w, r)
}

func findMux(host string, path PathKey) (*http.ServeMux, error) {
	sd := strings.Split(host, ".")[0]
	if r, ok := httpRoutes.UnsafeGet(PT.Alias(sd)); ok {
		return r.mux, nil
	}
	return nil, E.NotExists("route", fmt.Sprintf("subdomain: %s, path: %s", sd, path))
}

// TODO: default + per proxy
var (
	transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
	}

	transportNoTLS = func() *http.Transport {
		var clone = transport.Clone()
		clone.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return clone
	}()
)
