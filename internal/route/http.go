package route

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/api/v1/errorpage"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher"
	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	url "github.com/yusing/go-proxy/internal/net/types"
	P "github.com/yusing/go-proxy/internal/proxy"
	PT "github.com/yusing/go-proxy/internal/proxy/fields"
	"github.com/yusing/go-proxy/internal/types"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	HTTPRoute struct {
		*P.ReverseProxyEntry

		HealthMon health.HealthMonitor `json:"health,omitempty"`

		loadBalancer *loadbalancer.LoadBalancer
		server       *loadbalancer.Server
		handler      http.Handler
		rp           *gphttp.ReverseProxy
	}

	SubdomainKey = PT.Alias

	ReverseProxyHandler struct {
		*gphttp.ReverseProxy
	}
)

var (
	findMuxFunc = findMuxAnyDomain

	httpRoutes   = F.NewMapOf[string, *HTTPRoute]()
	httpRoutesMu sync.Mutex
	// globalMux    = http.NewServeMux() // TODO: support regex subdomain matching.
)

func (rp ReverseProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp.ReverseProxy.ServeHTTP(w, r)
}

func GetReverseProxies() F.Map[string, *HTTPRoute] {
	return httpRoutes
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

	if entry.NoTLSVerify {
		trans = gphttp.DefaultTransportNoTLS.Clone()
	} else {
		trans = gphttp.DefaultTransport.Clone()
	}

	rp := gphttp.NewReverseProxy(string(entry.Alias), entry.URL, trans)

	if len(entry.Middlewares) > 0 {
		err := middleware.PatchReverseProxy(string(entry.Alias), rp, entry.Middlewares)
		if err != nil {
			return nil, err
		}
	}

	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

	r := &HTTPRoute{
		ReverseProxyEntry: entry,
		rp:                rp,
	}
	return r, nil
}

func (r *HTTPRoute) String() string {
	return string(r.Alias)
}

func (r *HTTPRoute) URL() url.URL {
	return r.ReverseProxyEntry.URL
}

func (r *HTTPRoute) Start() E.NestedError {
	if r.ShouldNotServe() {
		return nil
	}

	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

	if r.handler != nil {
		return nil
	}

	if r.HealthCheck.Disabled && (r.UseIdleWatcher() || r.UseLoadBalance()) {
		logrus.Warnf("%s.healthCheck.disabled cannot be false when loadbalancer or idlewatcher is enabled", r.Alias)
		r.HealthCheck.Disabled = true
	}

	switch {
	case r.UseIdleWatcher():
		watcher, err := idlewatcher.Register(r.ReverseProxyEntry)
		if err != nil {
			return err
		}
		waker := idlewatcher.NewWaker(watcher, r.rp)
		r.handler = waker
		r.HealthMon = waker
	case !r.HealthCheck.Disabled:
		r.HealthMon = health.NewHTTPHealthMonitor(common.GlobalTask("Reverse proxy "+r.String()), r.URL(), r.HealthCheck)
	}

	if r.handler == nil {
		switch {
		case len(r.PathPatterns) == 1 && r.PathPatterns[0] == "/":
			r.handler = ReverseProxyHandler{r.rp}
		default:
			mux := http.NewServeMux()
			for _, p := range r.PathPatterns {
				mux.HandleFunc(string(p), r.rp.ServeHTTP)
			}
			r.handler = mux
		}
	}

	if r.HealthMon != nil {
		r.HealthMon.Start()
	}

	if r.UseLoadBalance() {
		r.addToLoadBalancer()
	} else {
		httpRoutes.Store(string(r.Alias), r)
	}

	return nil
}

func (r *HTTPRoute) Stop() (_ E.NestedError) {
	if r.handler == nil {
		return
	}

	httpRoutesMu.Lock()
	defer httpRoutesMu.Unlock()

	if r.loadBalancer != nil {
		r.removeFromLoadBalancer()
	} else {
		httpRoutes.Delete(string(r.Alias))
	}

	if r.HealthMon != nil {
		r.HealthMon.Stop()
		r.HealthMon = nil
	}

	r.handler = nil

	return
}

func (r *HTTPRoute) Started() bool {
	return r.handler != nil
}

func (r *HTTPRoute) addToLoadBalancer() {
	var lb *loadbalancer.LoadBalancer
	linked, ok := httpRoutes.Load(r.LoadBalance.Link)
	if ok {
		lb = linked.loadBalancer
		lb.UpdateConfigIfNeeded(r.LoadBalance)
		if linked.Raw.Homepage == nil && r.Raw.Homepage != nil {
			linked.Raw.Homepage = r.Raw.Homepage
		}
	} else {
		lb = loadbalancer.New(r.LoadBalance)
		lb.Start()
		linked = &HTTPRoute{
			ReverseProxyEntry: &P.ReverseProxyEntry{
				Raw: &types.RawEntry{
					Homepage: r.Raw.Homepage,
				},
				Alias: PT.Alias(lb.Link),
			},
			HealthMon:    lb,
			loadBalancer: lb,
			handler:      lb,
		}
		httpRoutes.Store(r.LoadBalance.Link, linked)
	}
	r.loadBalancer = lb
	r.server = loadbalancer.NewServer(string(r.Alias), r.rp.TargetURL, r.LoadBalance.Weight, r.handler, r.HealthMon)
	lb.AddServer(r.server)
}

func (r *HTTPRoute) removeFromLoadBalancer() {
	r.loadBalancer.RemoveServer(r.server)
	if r.loadBalancer.IsEmpty() {
		httpRoutes.Delete(r.LoadBalance.Link)
		logrus.Debugf("loadbalancer %q removed from route table", r.LoadBalance.Link)
	}
	r.server = nil
	r.loadBalancer = nil
}

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	mux, err := findMuxFunc(r.Host)
	// Why use StatusNotFound instead of StatusBadRequest or StatusBadGateway?
	// On nginx, when route for domain does not exist, it returns StatusBadGateway.
	// Then scraper / scanners will know the subdomain is invalid.
	// With StatusNotFound, they won't know whether it's the path, or the subdomain that is invalid.
	if err != nil {
		if !middleware.ServeStaticErrorPageFile(w, r) {
			logrus.Error(E.Failure("request").
				Subjectf("%s %s", r.Method, r.URL.String()).
				With(err))
			errorPage, ok := errorpage.GetErrorPageByStatus(http.StatusNotFound)
			if ok {
				w.WriteHeader(http.StatusNotFound)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if _, err := w.Write(errorPage); err != nil {
					logrus.Errorf("failed to respond error page to %s: %s", r.RemoteAddr, err)
				}
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
		return nil, errors.New("missing subdomain in url")
	}
	sd := strings.Join(hostSplit[:n-2], ".")
	if r, ok := httpRoutes.Load(sd); ok {
		return r.handler, nil
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
			return r.handler, nil
		}
		return nil, fmt.Errorf("no such route: %s", subdomain)
	}
}
