package loadbalancer

import (
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/task"
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
		zerolog.Logger

		impl
		*Config

		task task.Task

		pool   Pool
		poolMu sync.Mutex

		sumWeight weightType
		startTime time.Time
	}

	weightType uint16
)

const maxWeight weightType = 100

func New(cfg *Config) *LoadBalancer {
	lb := &LoadBalancer{
		Logger: logger.With().Str("name", cfg.Link).Logger(),
		Config: new(Config),
		pool:   newPool(),
	}
	lb.UpdateConfigIfNeeded(cfg)
	return lb
}

// Start implements task.TaskStarter.
func (lb *LoadBalancer) Start(routeSubtask task.Task) E.Error {
	lb.startTime = time.Now()
	lb.task = routeSubtask
	lb.task.OnFinished("loadbalancer cleanup", func() {
		if lb.impl != nil {
			lb.pool.RangeAll(func(k string, v *Server) {
				lb.impl.OnRemoveServer(v)
			})
		}
		lb.pool.Clear()
	})
	return nil
}

// Finish implements task.TaskFinisher.
func (lb *LoadBalancer) Finish(reason any) {
	lb.task.Finish(reason)
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
	lb.pool.RangeAll(func(_ string, srv *Server) {
		lb.impl.OnAddServer(srv)
	})
}

func (lb *LoadBalancer) UpdateConfigIfNeeded(cfg *Config) {
	if cfg != nil {
		lb.poolMu.Lock()
		defer lb.poolMu.Unlock()

		lb.Link = cfg.Link

		if lb.Mode == Unset && cfg.Mode != Unset {
			lb.Mode = cfg.Mode
			if !lb.Mode.ValidateUpdate() {
				lb.Error().Msgf("invalid mode %q, fallback to %q", cfg.Mode, lb.Mode)
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

	if lb.pool.Has(srv.Name) {
		old, _ := lb.pool.Load(srv.Name)
		lb.sumWeight -= old.Weight
		lb.impl.OnRemoveServer(old)
	}
	lb.pool.Store(srv.Name, srv)
	lb.sumWeight += srv.Weight

	lb.rebalance()
	lb.impl.OnAddServer(srv)

	lb.Debug().
		Str("action", "add").
		Str("server", srv.Name).
		Msgf("%d servers available", lb.pool.Size())
}

func (lb *LoadBalancer) RemoveServer(srv *Server) {
	lb.poolMu.Lock()
	defer lb.poolMu.Unlock()

	if !lb.pool.Has(srv.Name) {
		return
	}

	lb.pool.Delete(srv.Name)

	lb.sumWeight -= srv.Weight
	lb.rebalance()
	lb.impl.OnRemoveServer(srv)

	lb.Debug().
		Str("action", "remove").
		Str("server", srv.Name).
		Msgf("%d servers left", lb.pool.Size())

	if lb.pool.Size() == 0 {
		lb.task.Finish("no server left")
		return
	}
}

func (lb *LoadBalancer) rebalance() {
	if lb.sumWeight == maxWeight {
		return
	}
	if lb.pool.Size() == 0 {
		return
	}
	if lb.sumWeight == 0 { // distribute evenly
		weightEach := maxWeight / weightType(lb.pool.Size())
		remainder := maxWeight % weightType(lb.pool.Size())
		lb.pool.RangeAll(func(_ string, s *Server) {
			s.Weight = weightEach
			lb.sumWeight += weightEach
			if remainder > 0 {
				s.Weight++
				remainder--
			}
		})
		return
	}

	// scale evenly
	scaleFactor := float64(maxWeight) / float64(lb.sumWeight)
	lb.sumWeight = 0

	lb.pool.RangeAll(func(_ string, s *Server) {
		s.Weight = weightType(float64(s.Weight) * scaleFactor)
		lb.sumWeight += s.Weight
	})

	delta := maxWeight - lb.sumWeight
	if delta == 0 {
		return
	}
	lb.pool.Range(func(_ string, s *Server) bool {
		if delta == 0 {
			return false
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
		return true
	})
}

func (lb *LoadBalancer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	srvs := lb.availServers()
	if len(srvs) == 0 {
		http.Error(rw, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get(common.HeaderCheckRedirect) != "" {
		// wake all servers
		for _, srv := range srvs {
			// wake only if server implements Waker
			waker, ok := srv.handler.(idlewatcher.Waker)
			if ok {
				if err := waker.Wake(); err != nil {
					lb.Err(err).Msgf("failed to wake server %s", srv.Name)
				}
			}
		}
	}
	lb.impl.ServeHTTP(srvs, rw, r)
}

func (lb *LoadBalancer) Uptime() time.Duration {
	return time.Since(lb.startTime)
}

// MarshalJSON implements health.HealthMonitor.
func (lb *LoadBalancer) MarshalJSON() ([]byte, error) {
	extra := make(map[string]any)
	lb.pool.RangeAll(func(k string, v *Server) {
		extra[v.Name] = v.healthMon
	})

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
	if lb.pool.Size() == 0 {
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

func (lb *LoadBalancer) availServers() []*Server {
	avail := make([]*Server, 0, lb.pool.Size())
	lb.pool.RangeAll(func(_ string, srv *Server) {
		if srv.Status().Good() {
			avail = append(avail, srv)
		}
	})
	return avail
}
