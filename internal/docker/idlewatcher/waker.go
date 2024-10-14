package idlewatcher

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type Waker struct {
	*Watcher

	client *http.Client
	rp     *gphttp.ReverseProxy
}

func NewWaker(w *Watcher, rp *gphttp.ReverseProxy) *Waker {
	return &Waker{
		Watcher: w,
		client: &http.Client{
			Timeout:   1 * time.Second,
			Transport: rp.Transport,
		},
		rp: rp,
	}
}

func (w *Waker) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	shouldNext := w.wake(rw, r)
	if !shouldNext {
		return
	}
	w.rp.ServeHTTP(rw, r)
}

/* HealthMonitor interface */

func (w *Waker) Start() {}

func (w *Waker) Stop() {
	w.Unregister()
}

func (w *Waker) UpdateConfig(config health.HealthCheckConfig) {
	panic("use idlewatcher.Register instead")
}

func (w *Waker) Name() string {
	return w.String()
}

func (w *Waker) String() string {
	return string(w.Alias)
}

func (w *Waker) Status() health.Status {
	if w.ready.Load() {
		return health.StatusHealthy
	}
	if !w.ContainerRunning {
		return health.StatusNapping
	}
	return health.StatusStarting
}

func (w *Waker) Uptime() time.Duration {
	return 0
}

func (w *Waker) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":   w.Name(),
		"url":    w.URL,
		"status": w.Status(),
		"config": health.HealthCheckConfig{
			Interval: w.IdleTimeout,
			Timeout:  w.WakeTimeout,
		},
	})
}

/* End of HealthMonitor interface */

func (w *Waker) wake(rw http.ResponseWriter, r *http.Request) (shouldNext bool) {
	w.resetIdleTimer()

	// pass through if container is ready
	if w.ready.Load() {
		return true
	}

	ctx, cancel := context.WithTimeout(r.Context(), w.WakeTimeout)
	defer cancel()

	accept := gphttp.GetAccept(r.Header)
	acceptHTML := (r.Method == http.MethodGet && accept.AcceptHTML() || r.RequestURI == "/" && accept.IsEmpty())

	isCheckRedirect := r.Header.Get(headerCheckRedirect) != ""
	if !isCheckRedirect && acceptHTML {
		// Send a loading response to the client
		body := w.makeRespBody("%s waking up...", w.ContainerName)
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Header().Set("Content-Length", strconv.Itoa(len(body)))
		rw.Header().Add("Cache-Control", "no-cache")
		rw.Header().Add("Cache-Control", "no-store")
		rw.Header().Add("Cache-Control", "must-revalidate")
		if _, err := rw.Write(body); err != nil {
			w.l.Errorf("error writing http response: %s", err)
		}
		return
	}

	// wake the container and reset idle timer
	// also wait for another wake request
	w.wakeCh <- struct{}{}

	if <-w.wakeDone != nil {
		http.Error(rw, "Error sending wake request", http.StatusInternalServerError)
		return
	}

	// maybe another request came in while we were waiting for the wake
	if w.ready.Load() {
		if isCheckRedirect {
			rw.WriteHeader(http.StatusOK)
			return
		}
		return true
	}

	for {
		select {
		case <-ctx.Done():
			http.Error(rw, "Waking timed out", http.StatusGatewayTimeout)
			return
		default:
		}

		wakeReq, err := http.NewRequestWithContext(
			ctx,
			http.MethodHead,
			w.URL.String(),
			nil,
		)
		if err != nil {
			w.l.Errorf("new request err to %s: %s", r.URL, err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		wakeResp, err := w.client.Do(wakeReq)
		if err == nil && wakeResp.StatusCode != http.StatusServiceUnavailable {
			w.ready.Store(true)
			w.l.Debug("awaken")
			if isCheckRedirect {
				rw.WriteHeader(http.StatusOK)
				return
			}
			logrus.Infof("container %s is ready, passing through to %s", w.Alias, w.rp.TargetURL)
			return true
		}

		// retry until the container is ready or timeout
		time.Sleep(100 * time.Millisecond)
	}
}
