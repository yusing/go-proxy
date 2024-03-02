package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"sync"
)

type Routes struct {
	HTTPRoutes   map[string][]HTTPRoute  // subdomain/alias -> path
	StreamRoutes map[string]*StreamRoute // port -> target
}

var routes = Routes{
	HTTPRoutes:   make(map[string][]HTTPRoute),
	StreamRoutes: make(map[string]*StreamRoute),
}
var routesMutex = sync.Mutex{}

var streamSchemes = []string{"tcp", "udp"} // TODO: support "tcp:udp", "udp:tcp"
var httpSchemes = []string{"http", "https"}

var validSchemes = append(streamSchemes, httpSchemes...)

var lastFreePort int


func isValidScheme(scheme string) bool {
	for _, v := range validSchemes {
		if v == scheme {
			return true
		}
	}
	return false
}

func isStreamScheme(scheme string) bool {
	for _, v := range streamSchemes {
		if v == scheme {
			return true
		}
	}
	return false
}

func initProxyMaps() {
	routesMutex.Lock()
	defer routesMutex.Unlock()

	lastFreePort = 20000
	oldStreamRoutes := routes.StreamRoutes
	routes.StreamRoutes = make(map[string]*StreamRoute)
	routes.HTTPRoutes = make(map[string][]HTTPRoute)

	var wg sync.WaitGroup
	wg.Add(len(oldStreamRoutes))
	defer wg.Wait()

	for _, route := range oldStreamRoutes {
		go func(r *StreamRoute) {
			r.Cancel()
			wg.Done()
		}(route)
	}
}

func countProxies() int {
	return len(routes.HTTPRoutes) + len(routes.StreamRoutes)
}

func createProxy(config ProxyConfig) {
	if isStreamScheme(config.Scheme) {
		_, inMap := routes.StreamRoutes[config.Port]
		if inMap {
			log.Printf("[Build] Duplicated stream :%s, ignoring", config.Port)
			return
		}
		route, err := NewStreamRoute(config)
		if err != nil {
			log.Println(err)
			return
		}
		routes.StreamRoutes[config.Port] = route
		go route.listenStream()
	} else {
		_, inMap := routes.HTTPRoutes[config.Alias]
		if !inMap {
			routes.HTTPRoutes[config.Alias] = make([]HTTPRoute, 0)
		}
		url, err := url.Parse(fmt.Sprintf("%s://%s:%s", config.Scheme, config.Host, config.Port))
		if err != nil {
			log.Fatal(err)
		}
		routes.HTTPRoutes[config.Alias] = append(routes.HTTPRoutes[config.Alias], NewHTTPRoute(url, config.Path))
	}
}

func findFreePort() (int, error) {
	var portStr string
	var l net.Listener
	var err error = nil

	for lastFreePort <= 21000 {
		portStr = fmt.Sprintf(":%d", lastFreePort)
		l, err = net.Listen("tcp", portStr)
		lastFreePort++
		if err != nil {
			l.Close()
			return lastFreePort, nil
		}
	}
	l, err = net.Listen("tcp", ":0")
	if err != nil {
		return -1, fmt.Errorf("unable to find free port: %v", err)
	}
	// NOTE: may not be after 20000
	return l.Addr().(*net.TCPAddr).Port, nil
}
