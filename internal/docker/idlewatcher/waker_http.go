package idlewatcher

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

// ServeHTTP implements http.Handler
func (w *Watcher) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	shouldNext := w.wakeFromHTTP(rw, r)
	if !shouldNext {
		return
	}
	w.rp.ServeHTTP(rw, r)
}

func (w *Watcher) wakeFromHTTP(rw http.ResponseWriter, r *http.Request) (shouldNext bool) {
	w.resetIdleTimer()

	if r.Body != nil {
		defer r.Body.Close()
	}

	// pass through if container is already ready
	if w.ready.Load() {
		return true
	}

	accept := gphttp.GetAccept(r.Header)
	acceptHTML := (r.Method == http.MethodGet && accept.AcceptHTML() || r.RequestURI == "/" && accept.IsEmpty())

	isCheckRedirect := r.Header.Get(headerCheckRedirect) != ""
	if !isCheckRedirect && acceptHTML {
		// Send a loading response to the client
		body := w.makeLoadingPageBody()
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Header().Set("Content-Length", strconv.Itoa(len(body)))
		rw.Header().Add("Cache-Control", "no-cache")
		rw.Header().Add("Cache-Control", "no-store")
		rw.Header().Add("Cache-Control", "must-revalidate")
		rw.Header().Add("Connection", "close")
		if _, err := rw.Write(body); err != nil {
			w.l.Errorf("error writing http response: %s", err)
		}
		return
	}

	ctx, cancel := context.WithTimeoutCause(r.Context(), w.WakeTimeout, errors.New("wake timeout"))
	defer cancel()

	checkCancelled := func() bool {
		select {
		case <-w.task.Context().Done():
			w.l.Debugf("wake cancelled: %s", context.Cause(w.task.Context()))
			http.Error(rw, "Service unavailable", http.StatusServiceUnavailable)
			return true
		case <-ctx.Done():
			w.l.Debugf("wake cancelled: %s", context.Cause(ctx))
			http.Error(rw, "Waking timed out", http.StatusGatewayTimeout)
			return true
		default:
			return false
		}
	}

	if checkCancelled() {
		return false
	}

	w.l.Debug("wake signal received")
	err := w.wakeIfStopped()
	if err != nil {
		w.l.Error(E.FailWith("wake", err))
		http.Error(rw, "Error waking container", http.StatusInternalServerError)
		return
	}

	for {
		if checkCancelled() {
			return false
		}

		if w.Status() == health.StatusHealthy {
			w.resetIdleTimer()
			if isCheckRedirect {
				logrus.Debugf("container %s is ready, redirecting...", w.String())
				rw.WriteHeader(http.StatusOK)
				return
			}
			logrus.Infof("container %s is ready, passing through to %s", w.String(), w.hc.URL())
			return true
		}

		// retry until the container is ready or timeout
		time.Sleep(idleWakerCheckInterval)
	}
}
