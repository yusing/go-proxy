package main

import (
	"fmt"
	"strings"
)

type pathPoolMap struct {
	*SafeMap[string, *httpLoadBalancePool]
}

func newPathPoolMap() pathPoolMap {
	return pathPoolMap{
		NewSafeMap[string](NewHTTPLoadBalancePool),
	}
}

func (m pathPoolMap) Add(path string, route *HTTPRoute) {
	m.Ensure(path)
	m.Get(path).Add(route)
}

func (m pathPoolMap) FindMatch(pathGot string) (*HTTPRoute, error) {
	for pathWant, v := range m.m {
		if strings.HasPrefix(pathGot, pathWant) {
			return v.Pick(), nil
		}
	}
	return nil, fmt.Errorf("no matching route for path %s", pathGot)
}
