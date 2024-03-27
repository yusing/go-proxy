package main

import (
	"strings"
)

type pathPoolMap struct {
	SafeMap[string, *httpLoadBalancePool]
}

func newPathPoolMap() pathPoolMap {
	return pathPoolMap{NewSafeMapOf[pathPoolMap](NewHTTPLoadBalancePool)}
}

func (m pathPoolMap) Add(path string, route *HTTPRoute) {
	m.Ensure(path)
	m.Get(path).Add(route)
}

func (m pathPoolMap) FindMatch(pathGot string) (*HTTPRoute, NestedErrorLike) {
	for pathWant, v := range m.Iterator() {
		if strings.HasPrefix(pathGot, pathWant) {
			return v.Pick(), nil
		}
	}
	return nil, NewNestedError("no matching path").Subject(pathGot)
}
