package idlewatcher

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	P "github.com/yusing/go-proxy/internal/proxy"
	PT "github.com/yusing/go-proxy/internal/proxy/fields"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	W "github.com/yusing/go-proxy/internal/watcher"
)

type (
	Watcher struct {
		*P.ReverseProxyEntry

		client D.Client

		ready        atomic.Bool  // whether the site is ready to accept connection
		stopByMethod StopCallback // send a docker command w.r.t. `stop_method`

		wakeCh   chan struct{}
		wakeDone chan E.NestedError
		ticker   *time.Ticker

		ctx      context.Context
		cancel   context.CancelFunc
		refCount *sync.WaitGroup

		l logrus.FieldLogger
	}

	WakeDone     <-chan error
	WakeFunc     func() WakeDone
	StopCallback func() E.NestedError
)

var (
	mainLoopCtx    context.Context
	mainLoopCancel context.CancelFunc
	mainLoopWg     sync.WaitGroup

	watcherMap   = F.NewMapOf[string, *Watcher]()
	watcherMapMu sync.Mutex

	portHistoryMap = F.NewMapOf[PT.Alias, string]()

	newWatcherCh = make(chan *Watcher)

	logger = logrus.WithField("module", "idle_watcher")
)

func Register(entry *P.ReverseProxyEntry) (*Watcher, E.NestedError) {
	failure := E.Failure("idle_watcher register")

	if entry.IdleTimeout == 0 {
		return nil, failure.With(E.Invalid("idle_timeout", 0))
	}

	watcherMapMu.Lock()
	defer watcherMapMu.Unlock()

	key := entry.ContainerID

	if entry.URL.Port() != "0" {
		portHistoryMap.Store(entry.Alias, entry.URL.Port())
	}

	if w, ok := watcherMap.Load(key); ok {
		w.refCount.Add(1)
		w.ReverseProxyEntry = entry
		return w, nil
	}

	client, err := D.ConnectClient(entry.DockerHost)
	if err.HasError() {
		return nil, failure.With(err)
	}

	w := &Watcher{
		ReverseProxyEntry: entry,
		client:            client,
		refCount:          &sync.WaitGroup{},
		wakeCh:            make(chan struct{}, 1),
		wakeDone:          make(chan E.NestedError),
		ticker:            time.NewTicker(entry.IdleTimeout),
		l:                 logger.WithField("container", entry.ContainerName),
	}
	w.refCount.Add(1)
	w.stopByMethod = w.getStopCallback()

	watcherMap.Store(key, w)

	go func() {
		newWatcherCh <- w
	}()

	return w, nil
}

func (w *Watcher) Unregister() {
	w.refCount.Add(-1)
}

func Start() {
	logger.Debug("started")
	defer logger.Debug("stopped")

	mainLoopCtx, mainLoopCancel = context.WithCancel(context.Background())

	for {
		select {
		case <-mainLoopCtx.Done():
			return
		case w := <-newWatcherCh:
			w.l.Debug("registered")
			mainLoopWg.Add(1)
			go func() {
				w.watchUntilCancel()
				w.refCount.Wait() // wait for 0 ref count

				watcherMap.Delete(w.ContainerID)
				w.l.Debug("unregistered")
				mainLoopWg.Done()
			}()
		}
	}
}

func Stop() {
	mainLoopCancel()
	mainLoopWg.Wait()
}

func (w *Watcher) containerStop() error {
	return w.client.ContainerStop(w.ctx, w.ContainerID, container.StopOptions{
		Signal:  string(w.StopSignal),
		Timeout: &w.StopTimeout,
	})
}

func (w *Watcher) containerPause() error {
	return w.client.ContainerPause(w.ctx, w.ContainerID)
}

func (w *Watcher) containerKill() error {
	return w.client.ContainerKill(w.ctx, w.ContainerID, string(w.StopSignal))
}

func (w *Watcher) containerUnpause() error {
	return w.client.ContainerUnpause(w.ctx, w.ContainerID)
}

func (w *Watcher) containerStart() error {
	return w.client.ContainerStart(w.ctx, w.ContainerID, container.StartOptions{})
}

func (w *Watcher) containerStatus() (string, E.NestedError) {
	json, err := w.client.ContainerInspect(w.ctx, w.ContainerID)
	if err != nil {
		return "", E.FailWith("inspect container", err)
	}
	return json.State.Status, nil
}

func (w *Watcher) wakeIfStopped() E.NestedError {
	if w.ready.Load() || w.ContainerRunning {
		return nil
	}

	status, err := w.containerStatus()

	if err.HasError() {
		return err
	}
	// "created", "running", "paused", "restarting", "removing", "exited", or "dead"
	switch status {
	case "exited", "dead":
		return E.From(w.containerStart())
	case "paused":
		return E.From(w.containerUnpause())
	case "running":
		return nil
	default:
		return E.Unexpected("container state", status)
	}
}

func (w *Watcher) getStopCallback() StopCallback {
	var cb func() error
	switch w.StopMethod {
	case PT.StopMethodPause:
		cb = w.containerPause
	case PT.StopMethodStop:
		cb = w.containerStop
	case PT.StopMethodKill:
		cb = w.containerKill
	default:
		panic("should not reach here")
	}
	return func() E.NestedError {
		status, err := w.containerStatus()
		if err.HasError() {
			return err
		}
		if status != "running" {
			return nil
		}
		return E.From(cb())
	}
}

func (w *Watcher) resetIdleTimer() {
	w.ticker.Reset(w.IdleTimeout)
}

func (w *Watcher) watchUntilCancel() {
	defer close(w.wakeCh)

	w.ctx, w.cancel = context.WithCancel(mainLoopCtx)

	dockerWatcher := W.NewDockerWatcherWithClient(w.client)
	dockerEventCh, dockerEventErrCh := dockerWatcher.EventsWithOptions(w.ctx, W.DockerListOptions{
		Filters: W.NewDockerFilter(
			W.DockerFilterContainer,
			W.DockerrFilterContainer(w.ContainerID),
			W.DockerFilterStart,
			W.DockerFilterStop,
			W.DockerFilterDie,
			W.DockerFilterKill,
			W.DockerFilterPause,
			W.DockerFilterUnpause,
		),
	})
	defer w.ticker.Stop()
	defer w.client.Close()

	for {
		select {
		case <-w.ctx.Done():
			w.l.Debug("stopped")
			return
		case err := <-dockerEventErrCh:
			if err != nil && err.IsNot(context.Canceled) {
				w.l.Error(E.FailWith("docker watcher", err))
			}
		case e := <-dockerEventCh:
			switch {
			// create / start / unpause
			case e.Action.IsContainerWake():
				w.ContainerRunning = true
				w.resetIdleTimer()
				w.l.Info(e)
			default: // stop / pause / kil
				w.ContainerRunning = false
				w.ticker.Stop()
				w.ready.Store(false)
				w.l.Info(e)
			}
		case <-w.ticker.C:
			w.l.Debug("idle timeout")
			w.ticker.Stop()
			if err := w.stopByMethod(); err != nil && err.IsNot(context.Canceled) {
				w.l.Error(E.FailWith("stop", err).Extraf("stop method: %s", w.StopMethod))
			}
		case <-w.wakeCh:
			w.l.Debug("wake signal received")
			w.resetIdleTimer()
			err := w.wakeIfStopped()
			if err != nil {
				w.l.Error(E.FailWith("wake", err))
			}
			w.wakeDone <- err
		}
	}
}
