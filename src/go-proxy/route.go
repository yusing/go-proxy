package main

import (
	"fmt"
	"log"
	"net/url"
	"sync"
)

type Routes struct {
	HTTPRoutes   map[string][]HTTPRoute  // id -> path
	StreamRoutes map[string]*StreamRoute // id -> target
	Mutex        sync.Mutex
}

var routes = Routes{
	HTTPRoutes:   make(map[string][]HTTPRoute),
	StreamRoutes: make(map[string]*StreamRoute),
	Mutex:        sync.Mutex{},
}

var streamSchemes = []string{"tcp", "udp"} // TODO: support "tcp:udp", "udp:tcp"
var httpSchemes = []string{"http", "https"}

var validSchemes = append(streamSchemes, httpSchemes...)

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

func initRoutes() {
	routes.Mutex.Lock()
	defer routes.Mutex.Unlock()

	utils.resetPortsInUse()
	routes.StreamRoutes = make(map[string]*StreamRoute)
	routes.HTTPRoutes = make(map[string][]HTTPRoute)
}

func countRoutes() int {
	return len(routes.HTTPRoutes) + len(routes.StreamRoutes)
}

func createRoute(config *ProxyConfig) {
	if isStreamScheme(config.Scheme) {
		_, inMap := routes.StreamRoutes[config.id]
		if inMap {
			log.Printf("[Build] Duplicated stream %s, ignoring", config.id)
			return
		}
		route, err := NewStreamRoute(config)
		if err != nil {
			log.Println(err)
			return
		}
		routes.Mutex.Lock()
		routes.StreamRoutes[config.id] = route
		routes.Mutex.Unlock()
	} else {
		routes.Mutex.Lock()
		_, inMap := routes.HTTPRoutes[config.Alias]
		if !inMap {
			routes.HTTPRoutes[config.Alias] = make([]HTTPRoute, 0)
		}
		url, err := url.Parse(fmt.Sprintf("%s://%s:%s", config.Scheme, config.Host, config.Port))
		if err != nil {
			log.Fatal(err)
		}
		routes.HTTPRoutes[config.Alias] = append(routes.HTTPRoutes[config.Alias], NewHTTPRoute(url, config.Path))
		routes.Mutex.Unlock()
	}
}
