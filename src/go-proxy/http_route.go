package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"
)

type HTTPRoute struct {
	Url      *url.URL
	Path     string
	PathMode string
	Proxy    *httputil.ReverseProxy
}

func isValidProxyPathMode(mode string) bool {
	switch mode {
	case ProxyPathMode_Forward, ProxyPathMode_Sub, ProxyPathMode_RemovedPath:
		return true
	default:
		return false
	}
}

func NewHTTPRoute(config *ProxyConfig) (*HTTPRoute, error) {
	url, err := url.Parse(fmt.Sprintf("%s://%s:%s", config.Scheme, config.Host, config.Port))
	if err != nil {
		glog.Infoln(err)
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = transport

	if !isValidProxyPathMode(config.PathMode) {
		return nil, fmt.Errorf("invalid path mode: %s", config.PathMode)
	}

	route := &HTTPRoute{
		Url:      url,
		Path:     config.Path,
		Proxy:    proxy,
		PathMode: config.PathMode,
	}

	proxy.Director = nil

	initRewrite := func(pr *httputil.ProxyRequest) {
		pr.SetURL(url)
		pr.SetXForwarded()
	}
	rewrite := initRewrite

	switch {
	case config.Path == "", config.PathMode == ProxyPathMode_Forward:
		break
	case config.PathMode == ProxyPathMode_Sub:
		rewrite = func(pr *httputil.ProxyRequest) {
			initRewrite(pr)
			// disable compression
			pr.Out.Header.Set("Accept-Encoding", "identity")
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, config.Path)
		}
		route.Proxy.ModifyResponse = func(r *http.Response) error {
			contentType, ok := r.Header["Content-Type"]
			if !ok || len(contentType) == 0 {
				glog.Infof("unknown content type for %s", r.Request.URL.String())
				return nil
			}
			if !strings.HasPrefix(contentType[0], "text/html") {
				return nil
			}
			err := utils.respRemovePath(r, config.Path)
			if err != nil {
				err = fmt.Errorf("failed to remove path prefix %s: %v", config.Path, err)
				r.Status = err.Error()
				r.StatusCode = http.StatusInternalServerError
			}
			return err
		}
	default:
		rewrite = func(pr *httputil.ProxyRequest) {
			initRewrite(pr)
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, config.Path)
		}
	}

	if glog.V(3) {
		route.Proxy.Rewrite = func(pr *httputil.ProxyRequest) {
			rewrite(pr)
			r := pr.In
			glog.Infof("[Request] %s %s%s", r.Method, r.Host, r.URL.Path)
			glog.V(4).InfoDepthf(1, "Headers: %v", r.Header)
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
	routeMap, ok := routes.HTTPRoutes.UnsafeGet(subdomain)
	if !ok {
		return nil, fmt.Errorf("no matching route for subdomain %s", subdomain)
	}
	return routeMap.FindMatch(path)
}

func httpProxyHandler(w http.ResponseWriter, r *http.Request) {
	route, err := findHTTPRoute(r.Host, r.URL.Path)
	if err != nil {
		err = fmt.Errorf("[Request] failed %s %s%s, error: %v",
			r.Method,
			r.Host,
			r.URL.Path,
			err,
		)
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	route.Proxy.ServeHTTP(w, r)
}

// TODO: default + per proxy
var transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}).DialContext,
	MaxIdleConns:        1000,
	MaxIdleConnsPerHost: 1000,
}
