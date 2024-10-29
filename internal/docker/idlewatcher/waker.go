package idlewatcher

import (
	"sync/atomic"
	"time"

	. "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type waker struct {
	_ U.NoCopy

	rp     *gphttp.ReverseProxy
	stream net.Stream
	hc     health.HealthChecker

	ready atomic.Bool
}

const (
	idleWakerCheckInterval = 100 * time.Millisecond
	idleWakerCheckTimeout  = time.Second
)

// TODO: support stream

func newWaker(providerSubTask task.Task, entry entry.Entry, rp *gphttp.ReverseProxy, stream net.Stream) (Waker, E.Error) {
	hcCfg := entry.HealthCheckConfig()
	hcCfg.Timeout = idleWakerCheckTimeout

	waker := &waker{
		rp:     rp,
		stream: stream,
	}

	watcher, err := registerWatcher(providerSubTask, entry, waker)
	if err != nil {
		return nil, E.Errorf("register watcher: %w", err)
	}

	if rp != nil {
		waker.hc = health.NewHTTPHealthChecker(entry.TargetURL(), hcCfg, rp.Transport)
	} else if stream != nil {
		waker.hc = health.NewRawHealthChecker(entry.TargetURL(), hcCfg)
	} else {
		panic("both nil")
	}
	return watcher, nil
}

// lifetime should follow route provider
func NewHTTPWaker(providerSubTask task.Task, entry entry.Entry, rp *gphttp.ReverseProxy) (Waker, E.Error) {
	return newWaker(providerSubTask, entry, rp, nil)
}

func NewStreamWaker(providerSubTask task.Task, entry entry.Entry, stream net.Stream) (Waker, E.Error) {
	return newWaker(providerSubTask, entry, nil, stream)
}

// Start implements health.HealthMonitor.
func (w *Watcher) Start(routeSubTask task.Task) E.Error {
	routeSubTask.Finish("ignored")
	w.task.OnCancel("stop route", func() {
		routeSubTask.Parent().Finish(w.task.FinishCause())
	})
	return nil
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
	if !w.ContainerRunning {
		return health.StatusNapping
	}

	if w.ready.Load() {
		return health.StatusHealthy
	}

	healthy, _, err := w.hc.CheckHealth()
	switch {
	case err != nil:
		w.ready.Store(false)
		return health.StatusError
	case healthy:
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
	return (&health.JSONRepresentation{
		Name:   w.Name(),
		Status: w.Status(),
		Config: w.hc.Config(),
		URL:    url,
	}).MarshalJSON()
}
