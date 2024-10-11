package idlewatcher

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

type Waker struct {
	*Watcher

	client *http.Client
	rp     *gphttp.ReverseProxy
}

func NewWaker(w *Watcher, rp *gphttp.ReverseProxy) *Waker {
	orig := rp.ServeHTTP
	// workaround for stopped containers port become zero
	rp.ServeHTTP = func(rw http.ResponseWriter, r *http.Request) {
		if rp.TargetURL.Port() == "0" {
			port, ok := portHistoryMap.Load(w.Alias)
			if !ok {
				w.l.Errorf("port history not found for %s", w.Alias)
				http.Error(rw, "internal server error", http.StatusInternalServerError)
				return
			}
			rp.TargetURL.Host = fmt.Sprintf("%s:%v", rp.TargetURL.Hostname(), port)
		}
		orig(rw, r)
	}
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
	w.wake(w.rp.ServeHTTP, rw, r)
}

func (w *Waker) wake(next http.HandlerFunc, rw http.ResponseWriter, r *http.Request) {
	w.resetIdleTimer()

	// pass through if container is ready
	if w.ready.Load() {
		next(rw, r)
		return
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
		} else {
			next(rw, r)
		}
		return
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
			} else {
				next(rw, r)
			}
			return
		}

		// retry until the container is ready or timeout
		time.Sleep(100 * time.Millisecond)
	}
}
