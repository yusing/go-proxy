package loadbalancer

import (
	"hash/fnv"
	"net"
	"net/http"
)

type ipHash struct{ *LoadBalancer }

func (lb *LoadBalancer) newIPHash() impl  { return &ipHash{lb} }
func (ipHash) OnAddServer(srv *Server)    {}
func (ipHash) OnRemoveServer(srv *Server) {}

func (impl ipHash) ServeHTTP(_ servers, rw http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(rw, "Internal error", http.StatusInternalServerError)
		logger.Errorf("invalid remote address %s: %s", r.RemoteAddr, err)
		return
	}
	idx := hashIP(ip) % uint32(len(impl.pool))
	if !impl.pool[idx].available.Load() {
		http.Error(rw, "Service unavailable", http.StatusServiceUnavailable)
	}
	impl.pool[idx].handler.ServeHTTP(rw, r)
}

func hashIP(ip string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(ip))
	return h.Sum32()
}
