package main

import "sync/atomic"

type httpLoadBalancePool struct {
	pool        []*HTTPRoute
	curentIndex atomic.Int32
}

func NewHTTPLoadBalancePool() *httpLoadBalancePool {
	return &httpLoadBalancePool{
		pool: make([]*HTTPRoute, 0),
	}
}

func (p *httpLoadBalancePool) Add(route *HTTPRoute) {
	p.pool = append(p.pool, route)
}

func (p *httpLoadBalancePool) Iterator() []*HTTPRoute {
	return p.pool
}
