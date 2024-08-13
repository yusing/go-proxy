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
		Alias     PT.Alias      `json:"alias"`
		Subroutes HTTPSubroutes `json:"subroutes"`

		mux *http.ServeMux
	}

	HTTPSubroute struct {
		TargetURL URL     `json:"targetURL"`
		Path      PathKey `json:"path"`

		proxy *P.ReverseProxy
	}

	URL struct {
		*url.URL
	}
	PathKey       = string
	SubdomainKey  = string
	HTTPSubroutes = map[PathKey]HTTPSubroute
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
	var r *HTTPRoute
	r, ok := httpRoutes.UnsafeGet(entry.Alias.String())
	if !ok {
		r = &HTTPRoute{
			Alias:     entry.Alias,
			Subroutes: make(HTTPSubroutes),
			mux:       http.NewServeMux(),
		}
		httpRoutes.UnsafeSet(entry.Alias.String(), r)
	}

	path := entry.Path.String()
	if _, exists := r.Subroutes[path]; exists {
		httpRoutes.Unlock()
		return nil, E.Duplicated("path", path)
	}
	r.mux.HandleFunc(path, rp.ServeHTTP)
	if err := recover(); err != nil {
		httpRoutes.Unlock()
		switch t := err.(type) {
		case error:
			// NOTE: likely path pattern error
			return nil, E.From(t)
		default:
			return nil, E.From(fmt.Errorf("%v", t))
		}
	}

	sr := HTTPSubroute{
		TargetURL: URL{entry.URL},
		proxy:     rp,
		Path:      path,
	}

	rewrite := rp.Rewrite

	if logrus.GetLevel() == logrus.DebugLevel {
		l := logrus.WithField("alias", entry.Alias)

		sr.proxy.Rewrite = func(pr *P.ProxyRequest) {
			l.Debug("request URL: ", pr.In.Host, pr.In.URL.Path)
			l.Debug("request headers: ", pr.In.Header)
			rewrite(pr)
		}
	} else {
		sr.proxy.Rewrite = rewrite
	}

	r.Subroutes[path] = sr
	httpRoutes.Unlock()
	return r, E.Nil()
}

func (r *HTTPRoute) String() string {
	return fmt.Sprintf("%s (reverse proxy)", r.Alias)
}

func (r *HTTPRoute) Start() E.NestedError {
	httpRoutes.Set(r.Alias.String(), r)
	return E.Nil()
}

func (r *HTTPRoute) Stop() E.NestedError {
	httpRoutes.Delete(r.Alias.String())
	return E.Nil()
}

func (r *HTTPRoute) GetSubroute(path PathKey) (HTTPSubroute, bool) {
	sr, ok := r.Subroutes[path]
	return sr, ok
}

func (u URL) MarshalText() (text []byte, err error) {
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
	if r, ok := httpRoutes.UnsafeGet(sd); ok {
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
