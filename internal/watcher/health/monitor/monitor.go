package monitor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/metrics"
	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/notif"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	HealthCheckFunc func() (result *health.HealthCheckResult, err error)
	monitor         struct {
		service string
		config  *health.HealthCheckConfig
		url     U.AtomicValue[types.URL]

		status     U.AtomicValue[health.Status]
		lastResult *health.HealthCheckResult
		lastSeen   time.Time

		checkHealth HealthCheckFunc
		startTime   time.Time

		metric *metrics.Gauge

		task *task.Task
	}
)

var ErrNegativeInterval = errors.New("negative interval")

func newMonitor(url types.URL, config *health.HealthCheckConfig, healthCheckFunc HealthCheckFunc) *monitor {
	mon := &monitor{
		config:      config,
		checkHealth: healthCheckFunc,
		startTime:   time.Now(),
	}
	mon.url.Store(url)
	mon.status.Store(health.StatusHealthy)
	return mon
}

func (mon *monitor) ContextWithTimeout(cause string) (ctx context.Context, cancel context.CancelFunc) {
	if mon.task != nil {
		return context.WithTimeoutCause(mon.task.Context(), mon.config.Timeout, errors.New(cause))
	}
	return context.WithTimeoutCause(context.Background(), mon.config.Timeout, errors.New(cause))
}

// Start implements task.TaskStarter.
func (mon *monitor) Start(routeSubtask *task.Task) E.Error {
	mon.service = routeSubtask.Parent().Name()
	mon.task = routeSubtask

	if mon.config.Interval <= 0 {
		return E.From(ErrNegativeInterval)
	}

	if common.PrometheusEnabled {
		mon.metric = metrics.GetServiceMetrics().HealthStatus.With(metrics.HealthMetricLabels(mon.service))
	}

	go func() {
		logger := logging.With().Str("name", mon.service).Logger()

		defer func() {
			if mon.status.Load() != health.StatusError {
				mon.status.Store(health.StatusUnknown)
			}
			mon.task.Finish(nil)
			if mon.metric != nil {
				mon.metric.Reset()
			}
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
func (mon *monitor) Config() *health.HealthCheckConfig {
	return mon.config
}

// Status implements HealthMonitor.
func (mon *monitor) Status() health.Status {
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
	res := mon.lastResult
	return (&JSONRepresentation{
		Name:     mon.service,
		Config:   mon.config,
		Status:   mon.status.Load(),
		Started:  mon.startTime,
		Uptime:   mon.Uptime(),
		Latency:  res.Latency,
		LastSeen: mon.lastSeen,
		Detail:   res.Detail,
		URL:      mon.url.Load(),
	}).MarshalJSON()
}

func (mon *monitor) checkUpdateHealth() error {
	logger := logging.With().Str("name", mon.Name()).Logger()
	result, err := mon.checkHealth()
	if err != nil {
		defer mon.task.Finish(err)
		mon.status.Store(health.StatusError)
		if !errors.Is(err, context.Canceled) {
			return fmt.Errorf("check health: %w", err)
		}
		return nil
	}

	mon.lastResult = result
	var status health.Status
	if result.Healthy {
		status = health.StatusHealthy
		mon.lastSeen = time.Now()
	} else {
		status = health.StatusUnhealthy
	}
	if result.Healthy != (mon.status.Swap(status) == health.StatusHealthy) {
		extras := map[string]any{
			"Service Name": mon.service,
			"Last Seen":    strutils.FormatLastSeen(mon.lastSeen),
		}
		if !mon.url.Load().Nil() {
			extras["Service URL"] = mon.url.Load().String()
		}
		if result.Detail != "" {
			extras["Detail"] = result.Detail
		}
		if result.Healthy {
			logger.Info().Msg("service is up")
			extras["Ping"] = fmt.Sprintf("%d ms", result.Latency.Milliseconds())
			notif.Notify(&notif.LogMessage{
				Title:  "✅ Service is up ✅",
				Extras: extras,
				Color:  notif.Green,
			})
		} else {
			logger.Warn().Msg("service went down")
			notif.Notify(&notif.LogMessage{
				Title:  "❌ Service went down ❌",
				Extras: extras,
				Color:  notif.Red,
			})
		}
	}
	if mon.metric != nil {
		mon.metric.Set(float64(status))
	}

	return nil
}
