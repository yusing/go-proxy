package health

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/net/types"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	HealthMonitor interface {
		Start()
		Stop()
		IsHealthy() bool
		String() string
	}
	HealthCheckFunc func() (healthy bool, detail string, err error)
	monitor         struct {
		Name     string
		URL      types.URL
		Interval time.Duration

		healthy     atomic.Bool
		checkHealth HealthCheckFunc

		task   common.Task
		cancel context.CancelFunc
		done   chan struct{}

		mu sync.Mutex
	}
)

var monMap = F.NewMapOf[string, HealthMonitor]()

func newMonitor(task common.Task, url types.URL, config *HealthCheckConfig, healthCheckFunc HealthCheckFunc) *monitor {
	task, cancel := task.SubtaskWithCancel("Health monitor for %s", task.Name())
	mon := &monitor{
		Name:        task.Name(),
		URL:         url.JoinPath(config.Path),
		Interval:    config.Interval,
		checkHealth: healthCheckFunc,
		task:        task,
		cancel:      cancel,
		done:        make(chan struct{}),
	}
	mon.healthy.Store(true)
	return mon
}

func IsHealthy(name string) (healthy bool, ok bool) {
	mon, ok := monMap.Load(name)
	if !ok {
		return
	}
	return mon.IsHealthy(), true
}

func (mon *monitor) Start() {
	defer monMap.Store(mon.Name, mon)
	defer logger.Debugf("%s health monitor started", mon)

	go func() {
		defer close(mon.done)
		defer mon.task.Finished()

		ok := mon.checkUpdateHealth()
		if !ok {
			return
		}

		ticker := time.NewTicker(mon.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-mon.task.Context().Done():
				return
			case <-ticker.C:
				ok = mon.checkUpdateHealth()
				if !ok {
					return
				}
			}
		}
	}()
	logger.Debugf("health monitor %q started", mon)
}

func (mon *monitor) Stop() {
	defer logger.Debugf("%s health monitor stopped", mon)

	monMap.Delete(mon.Name)

	mon.mu.Lock()
	defer mon.mu.Unlock()

	if mon.cancel == nil {
		return
	}

	mon.cancel()
	<-mon.done

	mon.cancel = nil
}

func (mon *monitor) IsHealthy() bool {
	return mon.healthy.Load()
}

func (mon *monitor) String() string {
	return mon.Name
}

func (mon *monitor) checkUpdateHealth() (hasError bool) {
	healthy, detail, err := mon.checkHealth()
	if err != nil {
		mon.healthy.Store(false)
		if !errors.Is(err, context.Canceled) {
			logger.Errorf("server %q failed to check health: %s", mon, err)
		}
		mon.Stop()
		return false
	}
	if healthy != mon.healthy.Swap(healthy) {
		if healthy {
			logger.Infof("server %q is up", mon)
		} else {
			logger.Warnf("server %q is down: %s", mon, detail)
		}
	}

	return true
}
