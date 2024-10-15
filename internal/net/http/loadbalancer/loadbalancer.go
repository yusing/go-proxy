package loadbalancer

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/log"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/watcher/health"
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
		*Config

		pool   servers
		poolMu sync.Mutex

		sumWeight weightType
		startTime time.Time
	}

	weightType uint16
)

const maxWeight weightType = 100

func New(cfg *Config) *LoadBalancer {
	lb := &LoadBalancer{Config: new(Config), pool: make(servers, 0)}
	lb.UpdateConfigIfNeeded(cfg)
	return lb
}

func (lb *LoadBalancer) updateImpl() {
	switch lb.Mode {
	case Unset, RoundRobin:
		lb.impl = lb.newRoundRobin()
	case LeastConn:
		lb.impl = lb.newLeastConn()
	case IPHash:
		lb.impl = lb.newIPHash()
	default: // should happen in test only
		lb.impl = lb.newRoundRobin()
	}
	for _, srv := range lb.pool {
		lb.impl.OnAddServer(srv)
	}
}

func (lb *LoadBalancer) UpdateConfigIfNeeded(cfg *Config) {
	if cfg != nil {
		lb.poolMu.Lock()
		defer lb.poolMu.Unlock()

		lb.Link = cfg.Link

		if lb.Mode == Unset && cfg.Mode != Unset {
			lb.Mode = cfg.Mode
			if !lb.Mode.ValidateUpdate() {
				logger.Warnf("loadbalancer %s: invalid mode %q, fallback to %q", cfg.Link, cfg.Mode, lb.Mode)
			}
			lb.updateImpl()
		}

		if len(lb.Options) == 0 && len(cfg.Options) > 0 {
			lb.Options = cfg.Options
		}
	}

	if lb.impl == nil {
		lb.updateImpl()
	}
}

func (lb *LoadBalancer) AddServer(srv *Server) {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()

	lb.pool = append(lb.pool, srv)
	lb.sumWeight += srv.Weight

	lb.Rebalance()
	lb.impl.OnAddServer(srv)
	logger.Debugf("[add] loadbalancer %s: %d servers available", lb.Link, len(lb.pool))
}

func (lb *LoadBalancer) RemoveServer(srv *Server) {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()

	lb.sumWeight -= srv.Weight
	lb.Rebalance()
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
	if lb.sumWeight != 0 {
		log.Warnf("weighted mode not supported yet")
	}

	lb.startTime = time.Now()
	logger.Debugf("loadbalancer %s started", lb.Link)
}

func (lb *LoadBalancer) Stop() {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()
	lb.pool = nil

	logger.Debugf("loadbalancer %s stopped", lb.Link)
}

func (lb *LoadBalancer) Uptime() time.Duration {
	return time.Since(lb.startTime)
}

// MarshalJSON implements health.HealthMonitor.
func (lb *LoadBalancer) MarshalJSON() ([]byte, error) {
	extra := make(map[string]any)
	for _, v := range lb.pool {
		extra[v.Name] = v.healthMon
	}
	return (&health.JSONRepresentation{
		Name:    lb.Name(),
		Status:  lb.Status(),
		Started: lb.startTime,
		Uptime:  lb.Uptime(),
		Extra: map[string]any{
			"config": lb.Config,
			"pool":   extra,
		},
	}).MarshalJSON()
}

// Name implements health.HealthMonitor.
func (lb *LoadBalancer) Name() string {
	return lb.Link
}

// Status implements health.HealthMonitor.
func (lb *LoadBalancer) Status() health.Status {
	if len(lb.pool) == 0 {
		return health.StatusUnknown
	}
	if len(lb.availServers()) == 0 {
		return health.StatusUnhealthy
	}
	return health.StatusHealthy
}

// String implements health.HealthMonitor.
func (lb *LoadBalancer) String() string {
	return lb.Name()
}

func (lb *LoadBalancer) availServers() servers {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()

	avail := make(servers, 0, len(lb.pool))
	for _, s := range lb.pool {
		if s.Status().Bad() {
			continue
		}
		avail = append(avail, s)
	}
	return avail
}

// static HealthMonitor interface check
func (lb *LoadBalancer) _() health.HealthMonitor {
	return lb
}
