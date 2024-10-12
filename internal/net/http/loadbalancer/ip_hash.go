package loadbalancer

import (
	"hash/fnv"
	"net"
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
)

type ipHash struct {
	*LoadBalancer
	realIP *middleware.Middleware
}

func (lb *LoadBalancer) newIPHash() impl {
	impl := &ipHash{LoadBalancer: lb}
	if len(lb.Options) == 0 {
		return impl
	}
	var err E.NestedError
	impl.realIP, err = middleware.NewRealIP(lb.Options)
	if err != nil {
		logger.Errorf("loadbalancer %s invalid real_ip options: %s, ignoring", lb.Link, err)
	}
	return impl
}
func (ipHash) OnAddServer(srv *Server)    {}
func (ipHash) OnRemoveServer(srv *Server) {}

func (impl ipHash) ServeHTTP(_ servers, rw http.ResponseWriter, r *http.Request) {
	if impl.realIP != nil {
		impl.realIP.ModifyRequest(impl.serveHTTP, rw, r)
	} else {
		impl.serveHTTP(rw, r)
	}
}

func (impl ipHash) serveHTTP(rw http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(rw, "Internal error", http.StatusInternalServerError)
		logger.Errorf("invalid remote address %s: %s", r.RemoteAddr, err)
		return
	}
	idx := hashIP(ip) % uint32(len(impl.pool))
	if !impl.pool[idx].IsHealthy() {
		http.Error(rw, "Service unavailable", http.StatusServiceUnavailable)
	}
	impl.pool[idx].handler.ServeHTTP(rw, r)
}

func hashIP(ip string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(ip))
	return h.Sum32()
}
