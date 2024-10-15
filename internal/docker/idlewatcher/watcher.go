package idlewatcher

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/common"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	P "github.com/yusing/go-proxy/internal/proxy"
	PT "github.com/yusing/go-proxy/internal/proxy/fields"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	W "github.com/yusing/go-proxy/internal/watcher"
)

type (
	Watcher struct {
		*P.ReverseProxyEntry

		client D.Client

		ready        atomic.Bool  // whether the site is ready to accept connection
		stopByMethod StopCallback // send a docker command w.r.t. `stop_method`

		ticker *time.Ticker

		task   common.Task
		cancel context.CancelFunc

		refCount *U.RefCount

		l logrus.FieldLogger
	}

	WakeDone     <-chan error
	WakeFunc     func() WakeDone
	StopCallback func() E.NestedError
)

var (
	watcherMap   = F.NewMapOf[string, *Watcher]()
	watcherMapMu sync.Mutex

	portHistoryMap = F.NewMapOf[PT.Alias, string]()

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
		w.refCount.Add()
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
		refCount:          U.NewRefCounter(),
		ticker:            time.NewTicker(entry.IdleTimeout),
		l:                 logger.WithField("container", entry.ContainerName),
	}
	w.task, w.cancel = common.NewTaskWithCancel("Idlewatcher for %s", w.Alias)
	w.stopByMethod = w.getStopCallback()

	watcherMap.Store(key, w)

	go w.watchUntilCancel()

	return w, nil
}

func (w *Watcher) Unregister() {
	w.refCount.Sub()
}

func (w *Watcher) containerStop() error {
	return w.client.ContainerStop(w.task.Context(), w.ContainerID, container.StopOptions{
		Signal:  string(w.StopSignal),
		Timeout: &w.StopTimeout,
	})
}

func (w *Watcher) containerPause() error {
	return w.client.ContainerPause(w.task.Context(), w.ContainerID)
}

func (w *Watcher) containerKill() error {
	return w.client.ContainerKill(w.task.Context(), w.ContainerID, string(w.StopSignal))
}

func (w *Watcher) containerUnpause() error {
	return w.client.ContainerUnpause(w.task.Context(), w.ContainerID)
}

func (w *Watcher) containerStart() error {
	return w.client.ContainerStart(w.task.Context(), w.ContainerID, container.StartOptions{})
}

func (w *Watcher) containerStatus() (string, E.NestedError) {
	if !w.client.Connected() {
		return "", E.Failure("docker client closed")
	}
	json, err := w.client.ContainerInspect(w.task.Context(), w.ContainerID)
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
	dockerWatcher := W.NewDockerWatcherWithClient(w.client)
	dockerEventCh, dockerEventErrCh := dockerWatcher.EventsWithOptions(w.task.Context(), W.DockerListOptions{
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

	defer func() {
		w.cancel()
		w.ticker.Stop()
		w.client.Close()
		watcherMap.Delete(w.ContainerID)
		w.task.Finished()
	}()

	for {
		select {
		case <-w.task.Context().Done():
			w.l.Debug("stopped by context done")
			return
		case <-w.refCount.Zero():
			w.l.Debug("stopped by zero ref count")
			return
		case err := <-dockerEventErrCh:
			if err != nil && err.IsNot(context.Canceled) {
				w.l.Error(E.FailWith("docker watcher", err))
				return
			}
		case e := <-dockerEventCh:
			switch {
			// create / start / unpause
			case e.Action.IsContainerWake():
				w.ContainerRunning = true
				w.resetIdleTimer()
				w.l.Info("container awaken")
			case e.Action.IsContainerSleep(): // stop / pause / kil
				w.ContainerRunning = false
				w.ticker.Stop()
				w.ready.Store(false)
			default:
				w.l.Errorf("unexpected docker event: %s", e)
			}
		case <-w.ticker.C:
			w.l.Debug("idle timeout")
			w.ticker.Stop()
			if err := w.stopByMethod(); err != nil && err.IsNot(context.Canceled) {
				w.l.Error(E.FailWith("stop", err).Extraf("stop method: %s", w.StopMethod))
			} else {
				w.l.Info("stopped by idle timeout")
			}
		}
	}
}
