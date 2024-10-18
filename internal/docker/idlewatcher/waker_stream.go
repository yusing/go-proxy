package idlewatcher

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

// Setup implements types.Stream.
func (w *Watcher) Setup() error {
	return w.stream.Setup()
}

// Accept implements types.Stream.
func (w *Watcher) Accept() (conn types.StreamConn, err error) {
	conn, err = w.stream.Accept()
	// timeout means no connection is accepted
	var nErr *net.OpError
	ok := errors.As(err, &nErr)
	if ok && nErr.Timeout() {
		return
	}
	if err := w.wakeFromStream(); err != nil {
		return nil, err
	}
	return w.stream.Accept()
}

// CloseListeners implements types.Stream.
func (w *Watcher) CloseListeners() {
	w.stream.CloseListeners()
}

// Handle implements types.Stream.
func (w *Watcher) Handle(conn types.StreamConn) error {
	if err := w.wakeFromStream(); err != nil {
		return err
	}
	return w.stream.Handle(conn)
}

func (w *Watcher) wakeFromStream() error {
	// pass through if container is already ready
	if w.ready.Load() {
		return nil
	}

	w.l.Debug("wake signal received")
	wakeErr := w.wakeIfStopped()
	if wakeErr != nil {
		wakeErr = fmt.Errorf("wake failed with error: %w", wakeErr)
		w.l.Error(wakeErr)
		return wakeErr
	}

	ctx, cancel := context.WithTimeoutCause(w.task.Context(), w.WakeTimeout, errors.New("wake timeout"))
	defer cancel()

	for {
		select {
		case <-w.task.Context().Done():
			cause := w.task.FinishCause()
			w.l.Debugf("wake cancelled: %s", cause)
			return cause
		case <-ctx.Done():
			cause := context.Cause(ctx)
			w.l.Debugf("wake cancelled: %s", cause)
			return cause
		default:
		}

		if w.Status() == health.StatusHealthy {
			w.resetIdleTimer()
			logrus.Infof("container %s is ready, passing through to %s", w.String(), w.hc.URL())
			return nil
		}

		// retry until the container is ready or timeout
		time.Sleep(idleWakerCheckInterval)
	}
}
