package main

import (
	"fmt"

	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
)

/**
A small mod on net/http/httputil.ReverseProxy

Before mod:
root@http-benchmark-client:~# wrk -t 10 -c 200 -d 10s --latency http://bench.6uo.me/bench
Running 10s test @ http://bench.6uo.me/bench
  10 threads and 200 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     3.02ms    4.34ms 102.70ms   94.90%
    Req/Sec     8.06k     1.17k    9.99k    79.86%
  Latency Distribution
     50%    2.38ms
     75%    4.00ms
     90%    5.93ms
     99%   11.90ms
  808813 requests in 10.10s, 78.68MB read
Requests/sec:  80079.47
Transfer/sec:      7.79MB

After mod:
root@http-benchmark-client:~# wrk -t 10 -c 200 -d 10s --latency http://bench.6uo.me/bench
Running 10s test @ http://bench.6uo.me/bench
  10 threads and 200 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.77ms    5.64ms 118.14ms   99.07%
    Req/Sec    16.59k     2.22k   19.65k    87.30%
  Latency Distribution
     50%    1.11ms
     75%    1.85ms
     90%    2.74ms
     99%    6.68ms
  1665286 requests in 10.10s, 200.11MB read
Requests/sec: 164880.11
Transfer/sec:     19.81MB
**/
type HTTPRoute struct {
	Alias    string
	Url      *url.URL
	Path     string
	PathMode string
	Proxy    *ReverseProxy

	l logrus.FieldLogger
}

func NewHTTPRoute(config *ProxyConfig) (*HTTPRoute, error) {
	url, err := url.Parse(fmt.Sprintf("%s://%s:%s", config.Scheme, config.Host, config.Port))
	if err != nil {
		return nil, err
	}

	var tr *http.Transport
	if config.NoTLSVerify {
		tr = transportNoTLS
	} else {
		tr = transport
	}

	proxy := NewSingleHostReverseProxy(url, tr)

	if !isValidProxyPathMode(config.PathMode) {
		return nil, fmt.Errorf("invalid path mode: %s", config.PathMode)
	}

	route := &HTTPRoute{
		Alias:    config.Alias,
		Url:      url,
		Path:     config.Path,
		Proxy:    proxy,
		PathMode: config.PathMode,
		l: hrlog.WithFields(logrus.Fields{
			"alias":     config.Alias,
			"path":      config.Path,
			"path_mode": config.PathMode,
		}),
	}

	var rewrite func(*ProxyRequest)

	switch {
	case config.Path == "", config.PathMode == ProxyPathMode_Forward:
		rewrite = proxy.Rewrite
	case config.PathMode == ProxyPathMode_Sub:
		rewrite = func(pr *ProxyRequest) {
			proxy.Rewrite(pr)
			// disable compression
			pr.Out.Header.Set("Accept-Encoding", "identity")
			// remove path prefix
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, config.Path)
		}
		route.Proxy.ModifyResponse = func(r *http.Response) error {
			contentType, ok := r.Header["Content-Type"]
			if !ok || len(contentType) == 0 {
				route.l.Debug("unknown content type for", r.Request.URL.String())
				return nil
			}
			// disable cache
			r.Header.Set("Cache-Control", "no-store")

			var err error = nil
			switch {
			case strings.HasPrefix(contentType[0], "text/html"):
				err = utils.respHTMLSubPath(r, config.Path)
			case strings.HasPrefix(contentType[0], "application/javascript"):
				err = utils.respJSSubPath(r, config.Path)
			default:
				route.l.Debug("unknown content type(s): ", contentType)
			}
			if err != nil {
				err = fmt.Errorf("failed to remove path prefix %s: %v", config.Path, err)
				route.l.WithField("action", "path_sub").Error(err)
				r.Status = err.Error()
				r.StatusCode = http.StatusInternalServerError
			}
			return err
		}
	default:
		rewrite = func(pr *ProxyRequest) {
			proxy.Rewrite(pr)
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, config.Path)
		}
	}

	if logLevel == logrus.DebugLevel {
		route.Proxy.Rewrite = func(pr *ProxyRequest) {
			rewrite(pr)
			route.l.Debug("Request headers: ", pr.In.Header)
		}
	} else {
		route.Proxy.Rewrite = rewrite
	}

	return route, nil
}

func (p *httpLoadBalancePool) Pick() *HTTPRoute {
	// round-robin
	index := int(p.curentIndex.Load())
	defer p.curentIndex.Add(1)
	return p.pool[index%len(p.pool)]
}

func (r *HTTPRoute) RemoveFromRoutes() {
	httpRoutes.Delete(r.Alias)
}

// dummy implementation for Route interface
func (r *HTTPRoute) SetupListen()   {}
func (r *HTTPRoute) Listen()        {}
func (r *HTTPRoute) StopListening() {}

func isValidProxyPathMode(mode string) bool {
	switch mode {
	case ProxyPathMode_Forward, ProxyPathMode_Sub, ProxyPathMode_RemovedPath:
		return true
	default:
		return false
	}
}

func redirectToTLS(w http.ResponseWriter, r *http.Request) {
	// Redirect to the same host but with HTTPS
	var redirectCode int
	if r.Method == http.MethodGet {
		redirectCode = http.StatusMovedPermanently
	} else {
		redirectCode = http.StatusPermanentRedirect
	}
	http.Redirect(w, r, fmt.Sprintf("https://%s%s?%s", r.Host, r.URL.Path, r.URL.RawQuery), redirectCode)
}

func findHTTPRoute(host string, path string) (*HTTPRoute, error) {
	subdomain := strings.Split(host, ".")[0]
	routeMap, ok := httpRoutes.UnsafeGet(subdomain)
	if ok {
		return routeMap.FindMatch(path)
	}
	return nil, fmt.Errorf("no matching route for subdomain %s", subdomain)
}

func httpProxyHandler(w http.ResponseWriter, r *http.Request) {
	route, err := findHTTPRoute(r.Host, r.URL.Path)
	if err != nil {
		err = fmt.Errorf("request failed %s %s%s, error: %v",
			r.Method,
			r.Host,
			r.URL.Path,
			err,
		)
		logrus.Error(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	route.Proxy.ServeHTTP(w, r)
}

// alias -> (path -> routes)
type HTTPRoutes = SafeMap[string, *pathPoolMap]

var httpRoutes HTTPRoutes = NewSafeMap[string](newPathPoolMap)
