package main

import (
	"sync"

	"github.com/golang/glog"
)

type Routes struct {
	HTTPRoutes   *SafeMap[string, pathPoolMap] // id -> (path -> routes)
	StreamRoutes *SafeMap[string, StreamRoute] // id -> target
	Mutex        sync.Mutex
}

var routes = Routes{}

func isValidScheme(scheme string) bool {
	for _, v := range ValidSchemes {
		if v == scheme {
			return true
		}
	}
	return false
}

func isStreamScheme(scheme string) bool {
	for _, v := range StreamSchemes {
		if v == scheme {
			return true
		}
	}
	return false
}

func InitRoutes() {
	utils.resetPortsInUse()
	routes.HTTPRoutes = NewSafeMap[string](newPathPoolMap)
	routes.StreamRoutes = NewSafeMap[string, StreamRoute]()
}

func CountRoutes() int {
	return routes.HTTPRoutes.Size() + routes.StreamRoutes.Size()
}

func CreateRoute(config *ProxyConfig) {
	if isStreamScheme(config.Scheme) {
		if routes.StreamRoutes.Contains(config.id) {
			glog.Infof("[Build] Duplicated %s stream %s, ignoring", config.Scheme, config.id)
			return
		}
		route, err := NewStreamRoute(config)
		if err != nil {
			glog.Infoln(err)
			return
		}
		routes.StreamRoutes.Set(config.id, route)
	} else {
		routes.HTTPRoutes.Ensure(config.Alias)
		route, err := NewHTTPRoute(config)
		if err != nil {
			glog.Infoln(err)
			return
		}
		routes.HTTPRoutes.Get(config.Alias).Add(config.Path, route)
	}
}
