package idlewatcher

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

// ServeHTTP implements http.Handler.
func (w *Watcher) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	shouldNext := w.wakeFromHTTP(rw, r)
	if !shouldNext {
		return
	}
	select {
	case <-r.Context().Done():
		return
	default:
		w.rp.ServeHTTP(rw, r)
	}
}

func (w *Watcher) wakeFromHTTP(rw http.ResponseWriter, r *http.Request) (shouldNext bool) {
	w.resetIdleTimer()

	// pass through if container is already ready
	if w.ready.Load() {
		return true
	}

	if r.Body != nil {
		defer r.Body.Close()
	}

	accept := gphttp.GetAccept(r.Header)
	acceptHTML := (r.Method == http.MethodGet && accept.AcceptHTML() || r.RequestURI == "/" && accept.IsEmpty())

	isCheckRedirect := r.Header.Get(common.HeaderCheckRedirect) != ""
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
			w.Err(err).Msg("error writing http response")
		}
		return false
	}

	ctx, cancel := context.WithTimeoutCause(r.Context(), w.WakeTimeout, errors.New("wake timeout"))
	defer cancel()

	checkCanceled := func() (canceled bool) {
		select {
		case <-ctx.Done():
			w.WakeDebug().Str("cause", context.Cause(ctx).Error()).Msg("canceled")
			return true
		case <-w.task.Context().Done():
			w.WakeDebug().Str("cause", w.task.FinishCause().Error()).Msg("canceled")
			http.Error(rw, "Service unavailable", http.StatusServiceUnavailable)
			return true
		default:
			return false
		}
	}

	if checkCanceled() {
		return false
	}

	w.WakeTrace().Msg("signal received")
	err := w.wakeIfStopped()
	if err != nil {
		w.WakeError(err)
		http.Error(rw, "Error waking container", http.StatusInternalServerError)
		return false
	}

	for {
		if checkCanceled() {
			return false
		}

		if w.Status() == health.StatusHealthy {
			w.resetIdleTimer()
			if isCheckRedirect {
				w.Debug().Msgf("redirecting to %s ...", w.hc.URL())
				rw.WriteHeader(http.StatusOK)
				return false
			}
			w.Debug().Msgf("passing through to %s ...", w.hc.URL())
			return true
		}

		// retry until the container is ready or timeout
		time.Sleep(idleWakerCheckInterval)
	}
}
