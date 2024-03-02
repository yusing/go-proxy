package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type HTTPRoute struct {
	Url   *url.URL
	Path  string
	Proxy *httputil.ReverseProxy
}

// TODO: default + per proxy
var transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}).DialContext,
	MaxIdleConns:          1000,
	MaxIdleConnsPerHost:   1000,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
	ForceAttemptHTTP2:     true,
}

func NewHTTPRoute(Url *url.URL, Path string) HTTPRoute {
	proxy := httputil.NewSingleHostReverseProxy(Url)
	proxy.Transport = transport
	return HTTPRoute{Url: Url, Path: Path, Proxy: proxy}
}

func redirectToTLS(w http.ResponseWriter, r *http.Request) {
	// Redirect to the same host but with HTTPS
	log.Printf("[Redirect] redirecting to https")
	var redirectCode int
	if r.Method == http.MethodGet {
		redirectCode = http.StatusMovedPermanently
	} else {
		redirectCode = http.StatusPermanentRedirect
	}
	http.Redirect(w, r, fmt.Sprintf("https://%s%s?%s", r.Host, r.URL.Path, r.URL.RawQuery), redirectCode)
}

func httpProxyHandler(w http.ResponseWriter, r *http.Request) {
	route, err := findHTTPRoute(r.Host, r.URL.Path)
	if err != nil {
		log.Printf("[Request] failed %s %s%s, error: %v",
			r.Method,
			r.Host,
			r.URL.Path,
			err,
		)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	route.Proxy.ServeHTTP(w, r)
}
