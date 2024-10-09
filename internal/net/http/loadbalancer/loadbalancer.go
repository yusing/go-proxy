package loadbalancer

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/log"
	E "github.com/yusing/go-proxy/internal/error"
)

// TODO: stats of each server
// TODO: support weighted mode
type (
	impl interface {
		ServeHTTP(srvs servers, rw http.ResponseWriter, r *http.Request)
		OnAddServer(srv *Server)
		OnRemoveServer(srv *Server)
	}
	Config struct {
		Link   string
		Mode   Mode
		Weight weightType
	}
	LoadBalancer struct {
		impl
		Config

		pool   servers
		poolMu sync.RWMutex

		ctx    context.Context
		cancel context.CancelFunc
		done   chan struct{}

		sumWeight weightType
	}

	weightType uint16
)

const maxWeight weightType = 100

func New(cfg Config) *LoadBalancer {
	lb := &LoadBalancer{Config: cfg, pool: servers{}}
	mode := cfg.Mode
	if !cfg.Mode.ValidateUpdate() {
		logger.Warnf("%s: invalid loadbalancer mode: %s, fallback to %s", cfg.Link, mode, cfg.Mode)
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
}

func (lb *LoadBalancer) RemoveServer(srv *Server) {
	lb.poolMu.RLock()
	defer lb.poolMu.RUnlock()

	lb.impl.OnRemoveServer(srv)

	for i, s := range lb.pool {
		if s == srv {
			lb.pool = append(lb.pool[:i], lb.pool[i+1:]...)
			break
		}
	}
	if lb.IsEmpty() {
		lb.Stop()
	}
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
		remainer := maxWeight % weightType(len(lb.pool))
		for _, s := range lb.pool {
			s.Weight = weightEach
			lb.sumWeight += weightEach
			if remainer > 0 {
				s.Weight++
			}
			remainer--
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
	if lb.IsEmpty() {
		return
	}

	if lb.sumWeight != 0 && lb.sumWeight != maxWeight {
		msg := E.NewBuilder("loadbalancer %s total weight %d != %d", lb.Link, lb.sumWeight, maxWeight)
		for _, s := range lb.pool {
			msg.Addf("%s: %d", s.Name, s.Weight)
		}
		lb.Rebalance()
		inner := E.NewBuilder("After rebalancing")
		for _, s := range lb.pool {
			inner.Addf("%s: %d", s.Name, s.Weight)
		}
		msg.Addf("%s", inner)
		logger.Warn(msg)
	}

	if lb.sumWeight != 0 {
		log.Warnf("Weighted mode not supported yet")
	}

	switch lb.Mode {
	case RoundRobin:
		lb.impl = lb.newRoundRobin()
	case LeastConn:
		lb.impl = lb.newLeastConn()
	case IPHash:
		lb.impl = lb.newIPHash()
	}

	lb.done = make(chan struct{}, 1)
	lb.ctx, lb.cancel = context.WithCancel(context.Background())

	updateAll := func() {
		var wg sync.WaitGroup
		wg.Add(len(lb.pool))
		for _, s := range lb.pool {
			go func(s *Server) {
				defer wg.Done()
				s.checkUpdateAvail(lb.ctx)
			}(s)
		}
		wg.Wait()
	}

	go func() {
		defer lb.cancel()
		defer close(lb.done)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		updateAll()
		for {
			select {
			case <-lb.ctx.Done():
				return
			case <-ticker.C:
				lb.poolMu.RLock()
				updateAll()
				lb.poolMu.RUnlock()
			}
		}
	}()
}

func (lb *LoadBalancer) Stop() {
	if lb.impl == nil {
		return
	}

	lb.cancel()

	<-lb.done
	lb.pool = nil
}

func (lb *LoadBalancer) availServers() servers {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()

	avail := servers{}
	for _, s := range lb.pool {
		if s.available.Load() {
			avail = append(avail, s)
		}
	}
	return avail
}
