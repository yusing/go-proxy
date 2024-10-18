package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	HealthMonitor interface {
		task.TaskStarter
		task.TaskFinisher
		fmt.Stringer
		json.Marshaler
		Status() Status
		Uptime() time.Duration
		Name() string
	}
	HealthChecker interface {
		CheckHealth() (healthy bool, detail string, err error)
		URL() types.URL
		Config() *HealthCheckConfig
		UpdateURL(url types.URL)
	}
	HealthCheckFunc func() (healthy bool, detail string, err error)
	monitor         struct {
		service string
		config  *HealthCheckConfig
		url     U.AtomicValue[types.URL]

		status      U.AtomicValue[Status]
		checkHealth HealthCheckFunc
		startTime   time.Time

		task task.Task
	}
)

var monMap = F.NewMapOf[string, HealthMonitor]()

func newMonitor(url types.URL, config *HealthCheckConfig, healthCheckFunc HealthCheckFunc) *monitor {
	mon := &monitor{
		config:      config,
		checkHealth: healthCheckFunc,
		startTime:   time.Now(),
		task:        task.DummyTask(),
	}
	mon.url.Store(url)
	mon.status.Store(StatusHealthy)
	return mon
}

func Inspect(service string) (HealthMonitor, bool) {
	return monMap.Load(service)
}

func (mon *monitor) ContextWithTimeout(cause string) (ctx context.Context, cancel context.CancelFunc) {
	if mon.task != nil {
		return context.WithTimeoutCause(mon.task.Context(), mon.config.Timeout, errors.New(cause))
	} else {
		return context.WithTimeoutCause(context.Background(), mon.config.Timeout, errors.New(cause))
	}
}

// Start implements task.TaskStarter.
func (mon *monitor) Start(routeSubtask task.Task) E.NestedError {
	mon.service = routeSubtask.Parent().Name()
	mon.task = routeSubtask

	if mon.config.Interval <= 0 {
		return E.Invalid("interval", mon.config.Interval)
	}

	go func() {
		defer func() {
			if mon.status.Load() != StatusError {
				mon.status.Store(StatusUnknown)
			}
			mon.task.Finish(mon.task.FinishCause().Error())
		}()

		if err := mon.checkUpdateHealth(); err != nil {
			logger.Errorf("healthchecker %s failure: %s", mon.service, err)
			return
		}

		monMap.Store(mon.service, mon)
		defer monMap.Delete(mon.service)

		ticker := time.NewTicker(mon.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-mon.task.Context().Done():
				return
			case <-ticker.C:
				err := mon.checkUpdateHealth()
				if err != nil {
					logger.Errorf("healthchecker %s failure: %s", mon.service, err)
					return
				}
			}
		}
	}()
	return nil
}

// Finish implements task.TaskFinisher.
func (mon *monitor) Finish(reason string) {
	mon.task.Finish(reason)
}

// UpdateURL implements HealthChecker.
func (mon *monitor) UpdateURL(url types.URL) {
	mon.url.Store(url)
}

// URL implements HealthChecker.
func (mon *monitor) URL() types.URL {
	return mon.url.Load()
}

// Config implements HealthChecker.
func (mon *monitor) Config() *HealthCheckConfig {
	return mon.config
}

// Status implements HealthMonitor.
func (mon *monitor) Status() Status {
	return mon.status.Load()
}

// Uptime implements HealthMonitor.
func (mon *monitor) Uptime() time.Duration {
	return time.Since(mon.startTime)
}

// Name implements HealthMonitor.
func (mon *monitor) Name() string {
	if mon.task == nil {
		return ""
	}
	return mon.task.Name()
}

// String implements fmt.Stringer of HealthMonitor.
func (mon *monitor) String() string {
	return mon.Name()
}

// MarshalJSON implements json.Marshaler of HealthMonitor.
func (mon *monitor) MarshalJSON() ([]byte, error) {
	return (&JSONRepresentation{
		Name:    mon.service,
		Config:  mon.config,
		Status:  mon.status.Load(),
		Started: mon.startTime,
		Uptime:  mon.Uptime(),
		URL:     mon.url.Load(),
	}).MarshalJSON()
}

func (mon *monitor) checkUpdateHealth() E.NestedError {
	healthy, detail, err := mon.checkHealth()
	if err != nil {
		defer mon.task.Finish(err.Error())
		mon.status.Store(StatusError)
		if !errors.Is(err, context.Canceled) {
			return E.Failure("check health").With(err)
		}
		return nil
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

	return nil
}
