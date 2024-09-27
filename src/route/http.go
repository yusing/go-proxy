package route

import (
	"sync"

	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/common"
	"github.com/yusing/go-proxy/docker/idlewatcher"
	E "github.com/yusing/go-proxy/error"
	. "github.com/yusing/go-proxy/http"
	P "github.com/yusing/go-proxy/proxy"
	PT "github.com/yusing/go-proxy/proxy/fields"
	"github.com/yusing/go-proxy/route/middleware"
	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	HTTPRoute struct {
		Alias        PT.Alias        `json:"alias"`
		TargetURL    *URL            `json:"target_url"`
		PathPatterns PT.PathPatterns `json:"path_patterns"`

		entry   *P.ReverseProxyEntry
		mux     *http.ServeMux
		handler *ReverseProxy

		regIdleWatcher   func() E.NestedError
		unregIdleWatcher func()
	}

	URL          url.URL
	SubdomainKey = PT.Alias
)

var (
	findMuxFunc = findMuxAnyDomain

	httpRoutes   = F.NewMapOf[SubdomainKey, *HTTPRoute]()
	httpRoutesMu sync.Mutex
	globalMux    = http.NewServeMux() // TODO: support regex subdomain matching
)

func SetFindMuxDomains(domains []string) {
	if len(domains) == 0 {
		findMuxFunc = findMuxAnyDomain
	} else {
		findMuxFunc = findMuxByDomain(domains)
	}
}

func NewHTTPRoute(entry *P.ReverseProxyEntry) (*HTTPRoute, E.NestedError) {
	var trans *http.Transport
	var regIdleWatcher func() E.NestedError
	var unregIdleWatcher func()

	if entry.NoTLSVerify {
		trans = common.DefaultTransportNoTLS.Clone()
	} else {
		trans = common.DefaultTransport.Clone()
	}

	rp := NewReverseProxy(entry.URL, trans)

	if len(entry.Middlewares) > 0 {
		err := middleware.PatchReverseProxy(rp, entry.Middlewares)
		if err != nil {
			return nil, err
		}
	}

	if entry.UseIdleWatcher() {
		// allow time for response header up to `WakeTimeout`
		if entry.WakeTimeout > trans.ResponseHeaderTimeout {
			trans.ResponseHeaderTimeout = entry.WakeTimeout
		}
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
		return nil, E.Duplicated("HTTPRoute alias", entry.Alias)
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
	if r.mux != nil {
		return nil
	}

	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

	if r.regIdleWatcher != nil {
		if err := r.regIdleWatcher(); err.HasError() {
			r.unregIdleWatcher = nil
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
	if r.mux == nil {
		return nil
	}

	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

	if r.unregIdleWatcher != nil {
		r.unregIdleWatcher()
		r.unregIdleWatcher = nil
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
	mux, err := findMuxFunc(r.Host)
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

func findMuxAnyDomain(host string) (*http.ServeMux, E.NestedError) {
	hostSplit := strings.Split(host, ".")
	n := len(hostSplit)
	if n <= 2 {
		return nil, E.Missing("subdomain")
	}
	sd := strings.Join(hostSplit[:n-2], ".")
	if r, ok := httpRoutes.Load(PT.Alias(sd)); ok {
		return r.mux, nil
	}
	return nil, E.NotExist("route", sd)
}

func findMuxByDomain(domains []string) func(host string) (*http.ServeMux, E.NestedError) {
	return func(host string) (*http.ServeMux, E.NestedError) {
		var subdomain string
		for _, domain := range domains {
			subdomain = strings.TrimSuffix(subdomain, domain)
			if subdomain != domain {
				break
			}
		}
		if subdomain == "" { // not matched
			return nil, E.Invalid("host", host)
		}
		if r, ok := httpRoutes.Load(PT.Alias(subdomain)); ok {
			return r.mux, nil
		}
		return nil, E.NotExist("route", subdomain)
	}
}
