package health

import (
	"context"
	"encoding/json"
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
		config *HealthCheckConfig
		url    types.URL

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
	task, cancel := task.SubtaskWithCancel("Health monitor for %s", task.Name())
	mon := &monitor{
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

func Inspect(name string) (status Status, ok bool) {
	mon, ok := monMap.Load(name)
	if !ok {
		return
	}
	return mon.Status(), true
}

func (mon *monitor) Start() {
	defer monMap.Store(mon.task.Name(), mon)
	defer logger.Debugf("%s health monitor started", mon.String())

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
	logger.Debugf("health monitor %q started", mon.String())
}

func (mon *monitor) Stop() {
	defer logger.Debugf("%s health monitor stopped", mon.String())

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
	return json.Marshal(map[string]any{
		"name":    mon.Name(),
		"url":     mon.url,
		"status":  mon.status.Load(),
		"uptime":  mon.Uptime().String(),
		"started": mon.startTime.Unix(),
		"config":  mon.config,
	})
}

func (mon *monitor) checkUpdateHealth() (hasError bool) {
	healthy, detail, err := mon.checkHealth()
	if err != nil {
		mon.status.Store(StatusError)
		if !errors.Is(err, context.Canceled) {
			logger.Errorf("%s failed to check health: %s", mon.String(), err)
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
			logger.Infof("%s is up", mon.String())
		} else {
			logger.Warnf("%s is down: %s", mon.String(), detail)
		}
	}

	return true
}
