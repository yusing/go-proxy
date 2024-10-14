package loadbalancer

import (
	"net/http"
	"sync/atomic"
)

type roundRobin struct {
	index atomic.Uint32
}

func (*LoadBalancer) newRoundRobin() impl         { return &roundRobin{} }
func (lb *roundRobin) OnAddServer(srv *Server)    {}
func (lb *roundRobin) OnRemoveServer(srv *Server) {}

func (lb *roundRobin) ServeHTTP(srvs servers, rw http.ResponseWriter, r *http.Request) {
	index := lb.index.Add(1)
	srvs[index%uint32(len(srvs))].ServeHTTP(rw, r)
	if lb.index.Load() >= 2*uint32(len(srvs)) {
		lb.index.Store(0)
	}
}
