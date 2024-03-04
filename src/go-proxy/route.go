package main

import (
	"fmt"
	"log"
	"net/url"
	"sync"
)

type Routes struct {
	HTTPRoutes   *SafeMap[string, []HTTPRoute] // id -> path
	StreamRoutes *SafeMap[string, StreamRoute] // id -> target
	Mutex        sync.Mutex
}

var routes = Routes{}

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
	utils.resetPortsInUse()
	routes.HTTPRoutes = NewSafeMap[string, []HTTPRoute](
		func() []HTTPRoute {
			return make([]HTTPRoute, 0)
		},
	)
	routes.StreamRoutes = NewSafeMap[string, StreamRoute]()
}

func countRoutes() int {
	return routes.HTTPRoutes.Size() + routes.StreamRoutes.Size()
}

func createRoute(config *ProxyConfig) {
	if isStreamScheme(config.Scheme) {
		if routes.StreamRoutes.Contains(config.id) {
			log.Printf("[Build] Duplicated %s stream %s, ignoring", config.Scheme, config.id)
			return
		}
		route, err := NewStreamRoute(config)
		if err != nil {
			log.Println(err)
			return
		}
		routes.StreamRoutes.Set(config.id, route)
	} else {
		routes.HTTPRoutes.Ensure(config.Alias)
		url, err := url.Parse(fmt.Sprintf("%s://%s:%s", config.Scheme, config.Host, config.Port))
		if err != nil {
			log.Println(err)
			return
		}
		route := NewHTTPRoute(url, config.Path)
		routes.HTTPRoutes.Set(config.Alias, append(routes.HTTPRoutes.Get(config.Alias), route))
	}
}
