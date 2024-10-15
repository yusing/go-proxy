package health

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/net/types"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	HealthMonitor interface {
		Start()
		Stop()
		Status() Status
		Uptime() time.Duration
		Name() string
		String() string
		MarshalJSON() ([]byte, error)
	}
	HealthCheckFunc func() (healthy bool, detail string, err error)
	monitor         struct {
		service string
		config  *HealthCheckConfig
		url     types.URL

		status      U.AtomicValue[Status]
		checkHealth HealthCheckFunc
		startTime   time.Time

		task   common.Task
		cancel context.CancelFunc
		done   chan struct{}

		mu sync.Mutex
	}
)

var monMap = F.NewMapOf[string, HealthMonitor]()

func newMonitor(task common.Task, url types.URL, config *HealthCheckConfig, healthCheckFunc HealthCheckFunc) *monitor {
	service := task.Name()
	task, cancel := task.SubtaskWithCancel("Health monitor for %s", service)
	mon := &monitor{
		service:     service,
		config:      config,
		url:         url,
		checkHealth: healthCheckFunc,
		startTime:   time.Now(),
		task:        task,
		cancel:      cancel,
		done:        make(chan struct{}),
	}
	mon.status.Store(StatusHealthy)
	return mon
}

func Inspect(name string) (HealthMonitor, bool) {
	return monMap.Load(name)
}

func (mon *monitor) Start() {
	defer monMap.Store(mon.task.Name(), mon)

	go func() {
		defer close(mon.done)
		defer mon.task.Finished()

		ok := mon.checkUpdateHealth()
		if !ok {
			return
		}

		ticker := time.NewTicker(mon.config.Interval)
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
}

func (mon *monitor) Stop() {
	monMap.Delete(mon.task.Name())

	mon.mu.Lock()
	defer mon.mu.Unlock()

	if mon.cancel == nil {
		return
	}

	mon.cancel()
	<-mon.done

	mon.cancel = nil
	mon.status.Store(StatusUnknown)
}

func (mon *monitor) Status() Status {
	return mon.status.Load()
}

func (mon *monitor) Uptime() time.Duration {
	return time.Since(mon.startTime)
}

func (mon *monitor) Name() string {
	return mon.task.Name()
}

func (mon *monitor) String() string {
	return mon.Name()
}

func (mon *monitor) MarshalJSON() ([]byte, error) {
	return (&JSONRepresentation{
		Name:    mon.service,
		Config:  mon.config,
		Status:  mon.status.Load(),
		Started: mon.startTime,
		Uptime:  mon.Uptime(),
		URL:     mon.url,
	}).MarshalJSON()
}

func (mon *monitor) checkUpdateHealth() (hasError bool) {
	healthy, detail, err := mon.checkHealth()
	if err != nil {
		mon.status.Store(StatusError)
		if !errors.Is(err, context.Canceled) {
			logger.Errorf("%s failed to check health: %s", mon.service, err)
		}
		mon.Stop()
		return false
	}
	var status Status
	if healthy {
		status = StatusHealthy
	} else {
		status = StatusUnhealthy
	}
	if healthy != (mon.status.Swap(status) == StatusHealthy) {
		if healthy {
			logger.Infof("%s is up", mon.service)
		} else {
			logger.Warnf("%s is down: %s", mon.service, detail)
		}
	}

	return true
}
