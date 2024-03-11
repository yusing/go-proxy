package main

import (
	"fmt"
	"sync"
)

type Routes struct {
	HTTPRoutes   SafeMap[string, pathPoolMap] // alias -> (path -> routes)
	StreamRoutes SafeMap[string, StreamRoute] // id    -> target
	Mutex        sync.Mutex
}

type Route interface {
	SetupListen()
	Listen()
	StopListening()
	RemoveFromRoutes()
}

var routes = initRoutes()

func isValidScheme(s string) bool {
	for _, v := range ValidSchemes {
		if v == s {
			return true
		}
	}
	return false
}

func isStreamScheme(s string) bool {
	for _, v := range StreamSchemes {
		if v == s {
			return true
		}
	}
	return false
}

func initRoutes() *Routes {
	r := Routes{}
	r.HTTPRoutes = NewSafeMap[string](newPathPoolMap)
	r.StreamRoutes = NewSafeMap[string, StreamRoute]()
	return &r
}

func NewRoute(cfg *ProxyConfig) (Route, error) {
	if isStreamScheme(cfg.Scheme) {
		id := cfg.GetID()
		if routes.StreamRoutes.Contains(id) {
			return nil, fmt.Errorf("duplicated %s stream %s, ignoring", cfg.Scheme, id)
		}
		route, err := NewStreamRoute(cfg)
		if err != nil {
			return nil, err
		}
		routes.StreamRoutes.Set(id, route)
		return route, nil
	} else {
		routes.HTTPRoutes.Ensure(cfg.Alias)
		route, err := NewHTTPRoute(cfg)
		if err != nil {
			return nil, err
		}
		routes.HTTPRoutes.Get(cfg.Alias).Add(cfg.Path, route)
		return route, nil
	}
}