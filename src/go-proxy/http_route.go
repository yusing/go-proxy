package main

import (
	"fmt"

	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
)

type HTTPRoute struct {
	Alias    string
	Url      *url.URL
	Path     string
	PathMode string
	Proxy    *ReverseProxy

	l logrus.FieldLogger
}

func NewHTTPRoute(config *ProxyConfig) (*HTTPRoute, error) {
	u := fmt.Sprintf("%s://%s:%s", config.Scheme, config.Host, config.Port)
	url, err := url.Parse(u)
	if err != nil {
		return nil, NewNestedErrorf("invalid url").Subject(u).With(err)
	}

	var tr *http.Transport
	if config.NoTLSVerify {
		tr = transportNoTLS
	} else {
		tr = transport
	}

	proxy := NewSingleHostReverseProxy(url, tr)

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

	var rewriteBegin = proxy.Rewrite
	var rewrite func(*ProxyRequest)
	var modifyResponse func(*http.Response) error

	switch {
	case config.Path == "", config.PathMode == ProxyPathMode_Forward:
		rewrite = rewriteBegin
	case config.PathMode == ProxyPathMode_RemovedPath:
		rewrite = func(pr *ProxyRequest) {
			rewriteBegin(pr)
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, config.Path)
		}
	case config.PathMode == ProxyPathMode_Sub:
		rewrite = func(pr *ProxyRequest) {
			rewriteBegin(pr)
			// disable compression
			pr.Out.Header.Set("Accept-Encoding", "identity")
			// remove path prefix
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, config.Path)
		}
		modifyResponse = config.pathSubModResp
	default:
		return nil, NewNestedError("invalid path mode").Subject(config.PathMode)
	}

	if logLevel == logrus.DebugLevel {
		route.Proxy.Rewrite = func(pr *ProxyRequest) {
			rewrite(pr)
			route.l.Debug("request URL: ", pr.In.Host, pr.In.URL.Path)
			route.l.Debug("request headers: ", pr.In.Header)
		}
		route.Proxy.ModifyResponse = func(r *http.Response) error {
			route.l.Debug("response URL: ", r.Request.URL.String())
			route.l.Debug("response headers: ", r.Header)
			if modifyResponse != nil {
				return modifyResponse(r)
			}
			return nil
		}
	} else {
		route.Proxy.Rewrite = rewrite
	}

	return route, nil
}

func (r *HTTPRoute) Start() {
	// dummy
}
func (r *HTTPRoute) Stop() {
	httpRoutes.Delete(r.Alias)
}

func redirectToTLSHandler(w http.ResponseWriter, r *http.Request) {
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
	return nil, NewNestedError("no matching route for subdomain").Subject(subdomain)
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	route, err := findHTTPRoute(r.Host, r.URL.Path)
	if err != nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		err = NewNestedError("request failed").
			Subjectf("%s %s%s", r.Method, r.Host, r.URL.Path).
			With(err)
		logrus.Error(err)
		return
	}
	route.Proxy.ServeHTTP(w, r)
}

func (config *ProxyConfig) pathSubModResp(r *http.Response) error {
	contentType, ok := r.Header["Content-Type"]
	if !ok || len(contentType) == 0 {
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
	}
	if err != nil {
		err = NewNestedError("failed to remove path prefix").Subject(config.Path).With(err)
	}
	return err
}

// alias -> (path -> routes)
type HTTPRoutes = SafeMap[string, pathPoolMap]

var httpRoutes HTTPRoutes = NewSafeMapOf[HTTPRoutes](newPathPoolMap)
