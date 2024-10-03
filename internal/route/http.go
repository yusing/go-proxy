package route

import (
	"fmt"
	"sync"

	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/api/v1/error_page"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher"
	E "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	P "github.com/yusing/go-proxy/internal/proxy"
	PT "github.com/yusing/go-proxy/internal/proxy/fields"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	HTTPRoute struct {
		Alias        PT.Alias        `json:"alias"`
		TargetURL    *URL            `json:"target_url"`
		PathPatterns PT.PathPatterns `json:"path_patterns"`

		entry   *P.ReverseProxyEntry
		mux     http.Handler
		handler *ReverseProxy

		regIdleWatcher   func() E.NestedError
		unregIdleWatcher func()
	}

	URL          url.URL
	SubdomainKey = PT.Alias

	ReverseProxyHandler struct {
		*ReverseProxy
	}
)

var (
	findMuxFunc = findMuxAnyDomain

	httpRoutes   = F.NewMapOf[string, *HTTPRoute]()
	httpRoutesMu sync.Mutex
	globalMux    = http.NewServeMux() // TODO: support regex subdomain matching
)

func (rp ReverseProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp.ReverseProxy.ServeHTTP(w, r)
}

func SetFindMuxDomains(domains []string) {
	if len(domains) == 0 {
		findMuxFunc = findMuxAnyDomain
	} else {
		findMuxFunc = findMuxByDomains(domains)
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
		err := middleware.PatchReverseProxy(string(entry.Alias), rp, entry.Middlewares)
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
			idlewatcher.Unregister(entry)
			rp.Transport = trans
		}
	}

	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

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

	if r.entry.URL.Port() == "0" ||
		r.entry.IsDocker() && !r.entry.ContainerRunning {
		// TODO: if it use idlewatcher, set mux to dummy mux
		return nil
	}

	if len(r.PathPatterns) == 1 && r.PathPatterns[0] == "/" {
		r.mux = ReverseProxyHandler{r.handler}
	} else {
		mux := http.NewServeMux()
		for _, p := range r.PathPatterns {
			mux.HandleFunc(string(p), r.handler.ServeHTTP)
		}
		r.mux = mux
	}

	httpRoutes.Store(string(r.Alias), r)
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
	httpRoutes.Delete(string(r.Alias))
	return nil
}

func (r *HTTPRoute) Started() bool {
	return r.mux != nil
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
		if !middleware.ServeStaticErrorPageFile(w, r) {
			logrus.Error(E.Failure("request").
				Subjectf("%s %s", r.Method, r.URL.String()).
				With(err))
			errorPage, ok := error_page.GetErrorPageByStatus(http.StatusNotFound)
			if ok {
				w.WriteHeader(http.StatusNotFound)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(errorPage)
			} else {
				http.Error(w, err.Error(), http.StatusNotFound)
			}
		}
		return
	}
	mux.ServeHTTP(w, r)
}

func findMuxAnyDomain(host string) (http.Handler, error) {
	hostSplit := strings.Split(host, ".")
	n := len(hostSplit)
	if n <= 2 {
		return nil, fmt.Errorf("missing subdomain in url")
	}
	sd := strings.Join(hostSplit[:n-2], ".")
	if r, ok := httpRoutes.Load(sd); ok {
		return r.mux, nil
	}
	return nil, fmt.Errorf("no such route: %s", sd)
}

func findMuxByDomains(domains []string) func(host string) (http.Handler, error) {
	return func(host string) (http.Handler, error) {
		var subdomain string

		for _, domain := range domains {
			if !strings.HasPrefix(domain, ".") {
				domain = "." + domain
			}
			subdomain = strings.TrimSuffix(host, domain)
			if len(subdomain) < len(host) {
				break
			}
		}
		if len(subdomain) == len(host) { // not matched
			return nil, fmt.Errorf("%s does not match any base domain", host)
		}
		if r, ok := httpRoutes.Load(subdomain); ok {
			return r.mux, nil
		}
		return nil, fmt.Errorf("no such route: %s", subdomain)
	}
}
