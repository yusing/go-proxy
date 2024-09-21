package route

import (
	"crypto/tls"
	"net"
	"sync"
	"time"

	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/docker/idlewatcher"
	E "github.com/yusing/go-proxy/error"
	P "github.com/yusing/go-proxy/proxy"
	PT "github.com/yusing/go-proxy/proxy/fields"
	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	HTTPRoute struct {
		Alias        PT.Alias        `json:"alias"`
		TargetURL    *URL            `json:"target_url"`
		PathPatterns PT.PathPatterns `json:"path_patterns"`

		entry   *P.ReverseProxyEntry
		mux     *http.ServeMux
		handler *P.ReverseProxy

		regIdleWatcher   func() E.NestedError
		unregIdleWatcher func()
	}

	URL          url.URL
	SubdomainKey = PT.Alias
)

func NewHTTPRoute(entry *P.ReverseProxyEntry) (*HTTPRoute, E.NestedError) {
	var trans http.RoundTripper
	var regIdleWatcher func() E.NestedError
	var unregIdleWatcher func()

	if entry.NoTLSVerify {
		trans = transportNoTLS
	} else {
		trans = transport
	}

	rp := P.NewReverseProxy(entry.URL, trans, entry)

	if entry.UseIdleWatcher() {
		regIdleWatcher = func() E.NestedError {
			watcher, err := idlewatcher.Register(entry)
			if err.HasError() {
				return err
			}
			// patch round-tripper
			rp.Transport = watcher.PatchRoundTripper(trans)
			return nil
		}
		unregIdleWatcher = func() {
			idlewatcher.Unregister(entry.ContainerName)
			rp.Transport = trans
		}
	}

	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

	_, exists := httpRoutes.Load(entry.Alias)
	if exists {
		return nil, E.AlreadyExist("HTTPRoute alias", entry.Alias)
	}

	r := &HTTPRoute{
		Alias:            entry.Alias,
		TargetURL:        (*URL)(entry.URL),
		PathPatterns:     entry.PathPatterns,
		entry:            entry,
		handler:          rp,
		regIdleWatcher:   regIdleWatcher,
		unregIdleWatcher: unregIdleWatcher,
	}
	return r, nil
}

func (r *HTTPRoute) String() string {
	return string(r.Alias)
}

func (r *HTTPRoute) Start() E.NestedError {
	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

	if r.regIdleWatcher != nil {
		if err := r.regIdleWatcher(); err.HasError() {
			return err
		}
	}

	r.mux = http.NewServeMux()
	for _, p := range r.PathPatterns {
		r.mux.HandleFunc(string(p), r.handler.ServeHTTP)
	}

	httpRoutes.Store(r.Alias, r)
	return nil
}

func (r *HTTPRoute) Stop() E.NestedError {
	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

	if r.unregIdleWatcher != nil {
		r.unregIdleWatcher()
	}

	r.mux = nil
	httpRoutes.Delete(r.Alias)
	return nil
}

func (u *URL) String() string {
	return (*url.URL)(u).String()
}

func (u *URL) MarshalText() (text []byte, err error) {
	return []byte(u.String()), nil
}

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	mux, err := findMux(r.Host)
	if err != nil {
		err = E.Failure("request").
			Subjectf("%s %s%s", r.Method, r.Host, r.URL.Path).
			With(err)
		http.Error(w, err.String(), http.StatusNotFound)
		logrus.Error(err)
		return
	}
	mux.ServeHTTP(w, r)
}

func findMux(host string) (*http.ServeMux, E.NestedError) {
	sd := strings.Split(host, ".")[0]
	if r, ok := httpRoutes.Load(PT.Alias(sd)); ok {
		return r.mux, nil
	}
	return nil, E.NotExist("route", sd)
}

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

	httpRoutes   = F.NewMapOf[SubdomainKey, *HTTPRoute]()
	httpRoutesMu sync.Mutex
	globalMux    = http.NewServeMux()
)
