package idlewatcher

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	P "github.com/yusing/go-proxy/proxy"
	PT "github.com/yusing/go-proxy/proxy/fields"
	W "github.com/yusing/go-proxy/watcher"
)

type (
	watcher struct {
		*P.ReverseProxyEntry

		client D.Client

		ready        atomic.Bool  // whether the site is ready to accept connection
		stopByMethod StopCallback // send a docker command w.r.t. `stop_method`

		wakeCh   chan struct{}
		wakeDone chan E.NestedError

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

	watcherMap   = make(map[string]*watcher)
	watcherMapMu sync.Mutex

	newWatcherCh = make(chan *watcher)

	logger = logrus.WithField("module", "idle_watcher")
)

func Register(entry *P.ReverseProxyEntry) (*watcher, E.NestedError) {
	failure := E.Failure("idle_watcher register")

	if entry.IdleTimeout == 0 {
		return nil, failure.With(E.Invalid("idle_timeout", 0))
	}

	watcherMapMu.Lock()
	defer watcherMapMu.Unlock()

	if w, ok := watcherMap[entry.ContainerName]; ok {
		w.refCount.Add(1)
		w.ReverseProxyEntry = entry
		return w, nil
	}

	client, err := D.ConnectClient(entry.DockerHost)
	if err.HasError() {
		return nil, failure.With(err)
	}

	w := &watcher{
		ReverseProxyEntry: entry,
		client:            client,
		refCount:          &sync.WaitGroup{},
		wakeCh:            make(chan struct{}),
		wakeDone:          make(chan E.NestedError),
		l:                 logger.WithField("container", entry.ContainerName),
	}
	w.refCount.Add(1)
	w.stopByMethod = w.getStopCallback()

	watcherMap[w.ContainerName] = w

	go func() {
		newWatcherCh <- w
	}()

	return w, nil
}

func Unregister(containerName string) {
	if w, ok := watcherMap[containerName]; ok {
		w.refCount.Add(-1)
	}
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

				w.client.Close()
				delete(watcherMap, w.ContainerName)
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

func (w *watcher) PatchRoundTripper(rtp http.RoundTripper) roundTripper {
	return roundTripper{patched: func(r *http.Request) (*http.Response, error) {
		return w.roundTrip(rtp.RoundTrip, r)
	}}
}

func (w *watcher) containerStop() error {
	return w.client.ContainerStop(w.ctx, w.ContainerName, container.StopOptions{
		Signal:  string(w.StopSignal),
		Timeout: &w.StopTimeout})
}

func (w *watcher) containerPause() error {
	return w.client.ContainerPause(w.ctx, w.ContainerName)
}

func (w *watcher) containerKill() error {
	return w.client.ContainerKill(w.ctx, w.ContainerName, string(w.StopSignal))
}

func (w *watcher) containerUnpause() error {
	return w.client.ContainerUnpause(w.ctx, w.ContainerName)
}

func (w *watcher) containerStart() error {
	return w.client.ContainerStart(w.ctx, w.ContainerName, container.StartOptions{})
}

func (w *watcher) containerStatus() (string, E.NestedError) {
	json, err := w.client.ContainerInspect(w.ctx, w.ContainerName)
	if err != nil {
		return "", E.FailWith("inspect container", err)
	}
	return json.State.Status, nil
}

func (w *watcher) wakeIfStopped() E.NestedError {
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

func (w *watcher) getStopCallback() StopCallback {
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

func (w *watcher) watchUntilCancel() {
	defer close(w.wakeCh)

	w.ctx, w.cancel = context.WithCancel(context.Background())

	dockerWatcher := W.NewDockerWatcherWithClient(w.client)
	dockerEventCh, dockerEventErrCh := dockerWatcher.EventsWithOptions(w.ctx, W.DockerListOptions{
		Filters: W.NewDockerFilter(
			W.DockerFilterContainer,
			W.DockerrFilterContainerName(w.ContainerName),
			W.DockerFilterStart,
			W.DockerFilterStop,
			W.DockerFilterDie,
			W.DockerFilterKill,
			W.DockerFilterPause,
			W.DockerFilterUnpause,
		),
	})

	ticker := time.NewTicker(w.IdleTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-mainLoopCtx.Done():
			w.cancel()
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
				ticker.Reset(w.IdleTimeout)
				w.l.Info(e)
			default: // stop / pause / kill
				ticker.Stop()
				w.ready.Store(false)
				w.l.Info(e)
			}
		case <-ticker.C:
			w.l.Debug("idle timeout")
			ticker.Stop()
			if err := w.stopByMethod(); err != nil && err.IsNot(context.Canceled) {
				w.l.Error(E.FailWith("stop", err).Extraf("stop method: %s", w.StopMethod))
			}
		case <-w.wakeCh:
			w.l.Debug("wake signal received")
			ticker.Reset(w.IdleTimeout)
			err := w.wakeIfStopped()
			if err != nil && err.IsNot(context.Canceled) {
				w.l.Error(E.FailWith("wake", err))
			}
			select {
			case w.wakeDone <- err: // this is passed to roundtrip
			default:
			}
		}
	}
}
