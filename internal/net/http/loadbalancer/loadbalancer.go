package loadbalancer

import (
	"net/http"
	"sync"

	"github.com/go-acme/lego/v4/log"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
)

// TODO: stats of each server.
// TODO: support weighted mode.
type (
	impl interface {
		ServeHTTP(srvs servers, rw http.ResponseWriter, r *http.Request)
		OnAddServer(srv *Server)
		OnRemoveServer(srv *Server)
	}
	Config struct {
		Link    string                `json:"link" yaml:"link"`
		Mode    Mode                  `json:"mode" yaml:"mode"`
		Weight  weightType            `json:"weight" yaml:"weight"`
		Options middleware.OptionsRaw `json:"options,omitempty" yaml:"options,omitempty"`
	}
	LoadBalancer struct {
		impl
		Config

		pool   servers
		poolMu sync.Mutex

		sumWeight weightType
	}

	weightType uint16
)

const maxWeight weightType = 100

func New(cfg Config) *LoadBalancer {
	lb := &LoadBalancer{Config: cfg, pool: servers{}}
	mode := cfg.Mode
	if !cfg.Mode.ValidateUpdate() {
		logger.Warnf("loadbalancer %s: invalid mode %q, fallback to %s", cfg.Link, mode, cfg.Mode)
	}
	switch mode {
	case RoundRobin:
		lb.impl = lb.newRoundRobin()
	case LeastConn:
		lb.impl = lb.newLeastConn()
	case IPHash:
		lb.impl = lb.newIPHash()
	default: // should happen in test only
		lb.impl = lb.newRoundRobin()
	}
	return lb
}

func (lb *LoadBalancer) AddServer(srv *Server) {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()

	lb.pool = append(lb.pool, srv)
	lb.sumWeight += srv.Weight

	lb.impl.OnAddServer(srv)
	logger.Debugf("[add] loadbalancer %s: %d servers available", lb.Link, len(lb.pool))
}

func (lb *LoadBalancer) RemoveServer(srv *Server) {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()

	lb.impl.OnRemoveServer(srv)

	for i, s := range lb.pool {
		if s == srv {
			lb.pool = append(lb.pool[:i], lb.pool[i+1:]...)
			break
		}
	}
	if lb.IsEmpty() {
		lb.pool = nil
		return
	}

	lb.Rebalance()
	logger.Debugf("[remove] loadbalancer %s: %d servers left", lb.Link, len(lb.pool))
}

func (lb *LoadBalancer) IsEmpty() bool {
	return len(lb.pool) == 0
}

func (lb *LoadBalancer) Rebalance() {
	if lb.sumWeight == maxWeight {
		return
	}
	if lb.sumWeight == 0 { // distribute evenly
		weightEach := maxWeight / weightType(len(lb.pool))
		remainder := maxWeight % weightType(len(lb.pool))
		for _, s := range lb.pool {
			s.Weight = weightEach
			lb.sumWeight += weightEach
			if remainder > 0 {
				s.Weight++
				remainder--
			}
		}
		return
	}

	// scale evenly
	scaleFactor := float64(maxWeight) / float64(lb.sumWeight)
	lb.sumWeight = 0

	for _, s := range lb.pool {
		s.Weight = weightType(float64(s.Weight) * scaleFactor)
		lb.sumWeight += s.Weight
	}

	delta := maxWeight - lb.sumWeight
	if delta == 0 {
		return
	}
	for _, s := range lb.pool {
		if delta == 0 {
			break
		}
		if delta > 0 {
			s.Weight++
			lb.sumWeight++
			delta--
		} else {
			s.Weight--
			lb.sumWeight--
			delta++
		}
	}
}

func (lb *LoadBalancer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	srvs := lb.availServers()
	if len(srvs) == 0 {
		http.Error(rw, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	lb.impl.ServeHTTP(srvs, rw, r)
}

func (lb *LoadBalancer) Start() {
	if lb.sumWeight != 0 && lb.sumWeight != maxWeight {
		msg := E.NewBuilder("loadbalancer %s total weight %d != %d", lb.Link, lb.sumWeight, maxWeight)
		for _, s := range lb.pool {
			msg.Addf("%s: %d", s.Name, s.Weight)
		}
		lb.Rebalance()
		inner := E.NewBuilder("after rebalancing")
		for _, s := range lb.pool {
			inner.Addf("%s: %d", s.Name, s.Weight)
		}
		msg.Addf("%s", inner)
		logger.Warn(msg)
	}

	if lb.sumWeight != 0 {
		log.Warnf("weighted mode not supported yet")
	}
	logger.Debugf("loadbalancer %s started", lb.Link)
}

func (lb *LoadBalancer) Stop() {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()
	lb.pool = nil

	logger.Debugf("loadbalancer %s stopped", lb.Link)
}

func (lb *LoadBalancer) availServers() servers {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()

	avail := make(servers, 0, len(lb.pool))
	for _, s := range lb.pool {
		if s.IsHealthy() {
			avail = append(avail, s)
		}
	}
	return avail
}
