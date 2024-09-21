package idlewatcher

import (
	"bytes"
	"context"
	"io"
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
	event "github.com/yusing/go-proxy/watcher/events"
)

type watcher struct {
	*P.ReverseProxyEntry

	client D.Client

	refCount atomic.Int32

	stopByMethod StopCallback
	wakeCh       chan struct{}
	wakeDone     chan E.NestedError
	running      atomic.Bool

	ctx    context.Context
	cancel context.CancelFunc

	l logrus.FieldLogger
}

type (
	WakeDone     <-chan error
	WakeFunc     func() WakeDone
	StopCallback func() E.NestedError
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
		wakeCh:            make(chan struct{}, 1),
		wakeDone:          make(chan E.NestedError, 1),
		l:                 logger.WithField("container", entry.ContainerName),
	}
	w.refCount.Add(1)
	w.running.Store(entry.ContainerRunning)
	w.stopByMethod = w.getStopCallback()

	watcherMap[w.ContainerName] = w

	go func() {
		newWatcherCh <- w
	}()

	return w, nil
}

// If the container is not registered, this is no-op
func Unregister(containerName string) {
	watcherMapMu.Lock()
	defer watcherMapMu.Unlock()

	if w, ok := watcherMap[containerName]; ok {
		if w.refCount.Add(-1) > 0 {
			return
		}
		if w.cancel != nil {
			w.cancel()
		}
		w.client.Close()
		delete(watcherMap, containerName)
	}
}

func Start() {
	logger.Debug("started")
	defer logger.Debug("stopped")

	mainLoopCtx, mainLoopCancel = context.WithCancel(context.Background())

	defer mainLoopWg.Wait()

	for {
		select {
		case <-mainLoopCtx.Done():
			return
		case w := <-newWatcherCh:
			w.l.Debug("registered")
			mainLoopWg.Add(1)
			go func() {
				w.watch()
				Unregister(w.ContainerName)
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

func (w *watcher) roundTrip(origRoundTrip roundTripFunc, req *http.Request) (*http.Response, error) {
	w.wakeCh <- struct{}{}

	if w.running.Load() {
		return origRoundTrip(req)
	}
	timeout := time.After(w.WakeTimeout)

	for {
		if w.running.Load() {
			return origRoundTrip(req)
		}
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case err := <-w.wakeDone:
			if err != nil {
				return nil, err.Error()
			}
		case <-timeout:
			return getLoadingResponse(), nil
		}
	}
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
		w.running.Store(true)
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

func (w *watcher) watch() {
	watcherCtx, watcherCancel := context.WithCancel(context.Background())
	w.ctx = watcherCtx
	w.cancel = watcherCancel

	dockerWatcher := W.NewDockerWatcherWithClient(w.client)

	defer close(w.wakeCh)

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
		case <-watcherCtx.Done():
			w.l.Debug("stopped")
			return
		case err := <-dockerEventErrCh:
			if err != nil && err.IsNot(context.Canceled) {
				w.l.Error(E.FailWith("docker watcher", err))
			}
		case e := <-dockerEventCh:
			switch e.Action {
			case event.ActionDockerStartUnpause:
				w.running.Store(true)
				w.l.Infof("%s %s", e.ActorName, e.Action)
			case event.ActionDockerStopPause:
				w.running.Store(false)
				w.l.Infof("%s %s", e.ActorName, e.Action)
			}
		case <-ticker.C:
			w.l.Debug("timeout")
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

func getLoadingResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusAccepted,
		Header: http.Header{
			"Content-Type": {"text/html"},
			"Cache-Control": {
				"no-cache",
				"no-store",
				"must-revalidate",
			},
		},
		Body:          io.NopCloser(bytes.NewReader((loadingPage))),
		ContentLength: int64(len(loadingPage)),
	}
}

var (
	mainLoopCtx    context.Context
	mainLoopCancel context.CancelFunc
	mainLoopWg     sync.WaitGroup

	watcherMap   = make(map[string]*watcher)
	watcherMapMu sync.Mutex

	newWatcherCh = make(chan *watcher)

	logger = logrus.WithField("module", "idle_watcher")

	loadingPage = []byte(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Loading...</title>
</head>
<body>
	<script>
		window.onload = function() {
			setTimeout(function() {
				window.location.reload()
			}, 1000)
            // fetch(window.location.href)
			// 	.then(resp => resp.text())
			// 	.then(data => { document.body.innerHTML = data; })
			// 	.catch(err => { document.body.innerHTML = 'Error: ' + err; });
        };
	</script>
	<h1>Container is starting... Please wait</h1>
</body>
</html>
`[1:])
)
