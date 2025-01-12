package idlewatcher

import (
	"sync/atomic"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/metrics"
	"github.com/yusing/go-proxy/internal/net/http/reverseproxy"
	net "github.com/yusing/go-proxy/internal/net/types"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"
)

type (
	Waker = types.Waker
	waker struct {
		_ U.NoCopy

		rp     *reverseproxy.ReverseProxy
		stream net.Stream
		hc     health.HealthChecker
		metric *metrics.Gauge

		ready atomic.Bool
	}
)

const (
	idleWakerCheckInterval = 100 * time.Millisecond
	idleWakerCheckTimeout  = time.Second
)

// TODO: support stream

func newWaker(parent task.Parent, entry route.Entry, rp *reverseproxy.ReverseProxy, stream net.Stream) (Waker, E.Error) {
	hcCfg := entry.RawEntry().HealthCheck
	hcCfg.Timeout = idleWakerCheckTimeout

	waker := &waker{
		rp:     rp,
		stream: stream,
	}
	task := parent.Subtask("idlewatcher." + entry.TargetName())
	watcher, err := registerWatcher(task, entry, waker)
	if err != nil {
		return nil, E.Errorf("register watcher: %w", err)
	}

	switch {
	case rp != nil:
		waker.hc = monitor.NewHTTPHealthChecker(entry.TargetURL(), hcCfg)
	case stream != nil:
		waker.hc = monitor.NewRawHealthChecker(entry.TargetURL(), hcCfg)
	default:
		panic("both nil")
	}

	if common.PrometheusEnabled {
		m := metrics.GetServiceMetrics()
		fqn := parent.Name() + "/" + entry.TargetName()
		waker.metric = m.HealthStatus.With(metrics.HealthMetricLabels(fqn))
		waker.metric.Set(float64(watcher.Status()))
	}
	return watcher, nil
}

// lifetime should follow route provider.
func NewHTTPWaker(parent task.Parent, entry route.Entry, rp *reverseproxy.ReverseProxy) (Waker, E.Error) {
	return newWaker(parent, entry, rp, nil)
}

func NewStreamWaker(parent task.Parent, entry route.Entry, stream net.Stream) (Waker, E.Error) {
	return newWaker(parent, entry, nil, stream)
}

// Start implements health.HealthMonitor.
func (w *Watcher) Start(parent task.Parent) E.Error {
	w.task.OnCancel("route_cleanup", func() {
		parent.Finish(w.task.FinishCause())
		if w.metric != nil {
			w.metric.Reset()
		}
	})
	return nil
}

// Task implements health.HealthMonitor.
func (w *Watcher) Task() *task.Task {
	return w.task
}

// Finish implements health.HealthMonitor.
func (w *Watcher) Finish(reason any) {
	if w.stream != nil {
		w.stream.Close()
	}
}

// Name implements health.HealthMonitor.
func (w *Watcher) Name() string {
	return w.String()
}

// String implements health.HealthMonitor.
func (w *Watcher) String() string {
	return w.ContainerName
}

// Uptime implements health.HealthMonitor.
func (w *Watcher) Uptime() time.Duration {
	return 0
}

// Status implements health.HealthMonitor.
func (w *Watcher) Status() health.Status {
	status := w.getStatusUpdateReady()
	if w.metric != nil {
		w.metric.Set(float64(status))
	}
	return status
}

func (w *Watcher) getStatusUpdateReady() health.Status {
	if !w.ContainerRunning {
		return health.StatusNapping
	}

	if w.ready.Load() {
		return health.StatusHealthy
	}

	result, err := w.hc.CheckHealth()
	switch {
	case err != nil:
		w.ready.Store(false)
		return health.StatusError
	case result.Healthy:
		w.ready.Store(true)
		return health.StatusHealthy
	default:
		return health.StatusStarting
	}
}

// MarshalJSON implements health.HealthMonitor.
func (w *Watcher) MarshalJSON() ([]byte, error) {
	var url net.URL
	if w.hc.URL().Port() != "0" {
		url = w.hc.URL()
	}
	return (&monitor.JSONRepresentation{
		Name:   w.Name(),
		Status: w.Status(),
		Config: w.hc.Config(),
		URL:    url,
	}).MarshalJSON()
}
