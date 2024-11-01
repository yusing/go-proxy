package health

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/notif"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
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

var ErrNegativeInterval = errors.New("negative interval")

func newMonitor(url types.URL, config *HealthCheckConfig, healthCheckFunc HealthCheckFunc) *monitor {
	mon := &monitor{
		config:      config,
		checkHealth: healthCheckFunc,
		startTime:   time.Now(),
	}
	mon.url.Store(url)
	mon.status.Store(StatusHealthy)
	return mon
}

func (mon *monitor) ContextWithTimeout(cause string) (ctx context.Context, cancel context.CancelFunc) {
	if mon.task != nil {
		return context.WithTimeoutCause(mon.task.Context(), mon.config.Timeout, errors.New(cause))
	}
	return context.WithTimeoutCause(context.Background(), mon.config.Timeout, errors.New(cause))
}

// Start implements task.TaskStarter.
func (mon *monitor) Start(routeSubtask task.Task) E.Error {
	mon.service = routeSubtask.Parent().Name()
	mon.task = routeSubtask

	if mon.config.Interval <= 0 {
		return E.From(ErrNegativeInterval)
	}

	go func() {
		logger := logging.With().Str("name", mon.service).Logger()

		defer func() {
			if mon.status.Load() != StatusError {
				mon.status.Store(StatusUnknown)
			}
			mon.task.Finish(nil)
		}()

		if err := mon.checkUpdateHealth(); err != nil {
			logger.Err(err).Msg("healthchecker failure")
			return
		}

		ticker := time.NewTicker(mon.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-mon.task.Context().Done():
				return
			case <-ticker.C:
				err := mon.checkUpdateHealth()
				if err != nil {
					logger.Err(err).Msg("healthchecker failure")
					return
				}
			}
		}
	}()
	return nil
}

// Finish implements task.TaskFinisher.
func (mon *monitor) Finish(reason any) {
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
	parts := strings.Split(mon.service, "/")
	return parts[len(parts)-1]
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

func (mon *monitor) checkUpdateHealth() error {
	logger := logging.With().Str("name", mon.Name()).Logger()
	healthy, detail, err := mon.checkHealth()
	if err != nil {
		defer mon.task.Finish(err)
		mon.status.Store(StatusError)
		if !errors.Is(err, context.Canceled) {
			return fmt.Errorf("check health: %w", err)
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
			logger.Info().Msg("server is up")
			notif.Notify(mon.service, "server is up")
		} else {
			logger.Warn().Msg("server is down")
			logger.Debug().Msg(detail)
			notif.Notify(mon.service, "server is down")
		}
	}

	return nil
}
