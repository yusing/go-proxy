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
)

type watcher struct {
	*P.ReverseProxyEntry
	client D.Client

	refCount atomic.Int32

	stopByMethod StopCallback
	wakeCh       chan struct{}
	wakeDone     chan E.NestedError

	ctx    context.Context
	cancel context.CancelFunc

	l logrus.FieldLogger
}

type (
	WakeDone     <-chan error
	WakeFunc     func() WakeDone
	StopCallback func() (bool, E.NestedError)
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
		if w.refCount.Load() == 0 {
			w.cancel()
			close(w.wakeCh)
			delete(watcherMap, containerName)
		} else {
			w.refCount.Add(-1)
		}
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
	timeout := time.After(w.WakeTimeout)
	w.wakeCh <- struct{}{}
	for {
		select {
		case err := <-w.wakeDone:
			if err != nil {
				return nil, err.Error()
			}
			return origRoundTrip(req)
		case <-timeout:
			resp := loadingResponse
			resp.TLS = req.TLS
			return &resp, nil
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

func (w *watcher) wakeIfStopped() (bool, E.NestedError) {
	failure := E.Failure("wake")
	status, err := w.containerStatus()

	if err.HasError() {
		return false, failure.With(err)
	}
	// "created", "running", "paused", "restarting", "removing", "exited", or "dead"
	switch status {
	case "exited", "dead":
		err = E.From(w.containerStart())
	case "paused":
		err = E.From(w.containerUnpause())
	case "running":
		return false, nil
	default:
		return false, failure.With(E.Unexpected("container state", status))
	}

	if err.HasError() {
		return false, failure.With(err)
	}

	status, err = w.containerStatus()
	if err.HasError() {
		return false, failure.With(err)
	} else if status != "running" {
		return false, failure.With(E.Unexpected("container state", status))
	} else {
		return true, nil
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
	return func() (bool, E.NestedError) {
		status, err := w.containerStatus()
		if err.HasError() {
			return false, E.FailWith("stop", err)
		}
		if status != "running" {
			return false, nil
		}
		err = E.From(cb())
		if err.HasError() {
			return false, E.FailWith("stop", err)
		}
		return true, nil
	}
}

func (w *watcher) watch() {
	watcherCtx, watcherCancel := context.WithCancel(context.Background())
	w.ctx = watcherCtx
	w.cancel = watcherCancel

	ticker := time.NewTicker(w.IdleTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-mainLoopCtx.Done():
			watcherCancel()
		case <-watcherCtx.Done():
			w.l.Debug("stopped")
			return
		case <-ticker.C:
			w.l.Debug("timeout")
			stopped, err := w.stopByMethod()
			if err.HasError() {
				w.l.Error(err.Extraf("stop method: %s", w.StopMethod))
			} else if stopped {
				w.l.Infof("%s: ok", w.StopMethod)
			} else {
				ticker.Stop()
			}
		case <-w.wakeCh:
			w.l.Debug("wake received")
			go func() {
				started, err := w.wakeIfStopped()
				if err != nil {
					w.l.Error(err)
				} else if started {
					w.l.Infof("awaken")
					ticker.Reset(w.IdleTimeout)
				}
				w.wakeDone <- err // this is passed to roundtrip
			}()
		}
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

	loadingResponse = http.Response{
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
                location.reload();
            }, 1000); // 1000 milliseconds = 1 second
        };
	</script>
	<p>Container is starting... Please wait</p>
</body>
</html>
`[1:])
)
