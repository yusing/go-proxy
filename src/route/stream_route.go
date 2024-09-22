package route

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/error"
	P "github.com/yusing/go-proxy/proxy"
)

type StreamRoute struct {
	P.StreamEntry
	StreamImpl `json:"-"`

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	connCh  chan any
	started atomic.Bool
	l       logrus.FieldLogger
}

type StreamImpl interface {
	Setup() error
	Accept() (any, error)
	Handle(any) error
	CloseListeners()
}

func NewStreamRoute(entry *P.StreamEntry) (*StreamRoute, E.NestedError) {
	// TODO: support non-coherent scheme
	if !entry.Scheme.IsCoherent() {
		return nil, E.Unsupported("scheme", fmt.Sprintf("%v -> %v", entry.Scheme.ListeningScheme, entry.Scheme.ProxyScheme))
	}
	base := &StreamRoute{
		StreamEntry: *entry,
		connCh:      make(chan any, 100),
	}
	if entry.Scheme.ListeningScheme.IsTCP() {
		base.StreamImpl = NewTCPRoute(base)
	} else {
		base.StreamImpl = NewUDPRoute(base)
	}
	base.l = logrus.WithField("route", base.StreamImpl)
	return base, nil
}

func (r *StreamRoute) String() string {
	return fmt.Sprintf("%s stream: %s", r.Scheme, r.Alias)
}

func (r *StreamRoute) Start() E.NestedError {
	if r.started.Load() {
		return nil
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.wg.Wait()
	if err := r.Setup(); err != nil {
		return E.FailWith("setup", err)
	}
	r.started.Store(true)
	r.wg.Add(2)
	go r.grAcceptConnections()
	go r.grHandleConnections()
	return nil
}

func (r *StreamRoute) Stop() E.NestedError {
	if !r.started.Load() {
		return nil
	}
	l := r.l
	r.cancel()
	r.CloseListeners()

	done := make(chan struct{}, 1)
	go func() {
		r.wg.Wait()
		close(done)
	}()

	timeout := time.After(streamStopListenTimeout)
	for {
		select {
		case <-done:
			l.Debug("stopped listening")
			return nil
		case <-timeout:
			return E.FailedWhy("stop", "timed out")
		}
	}
}

func (r *StreamRoute) grAcceptConnections() {
	defer r.wg.Done()

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			conn, err := r.Accept()
			if err != nil {
				select {
				case <-r.ctx.Done():
					return
				default:
					r.l.Error(err)
					continue
				}
			}
			r.connCh <- conn
		}
	}
}

func (r *StreamRoute) grHandleConnections() {
	defer r.wg.Done()

	for {
		select {
		case <-r.ctx.Done():
			return
		case conn := <-r.connCh:
			go func() {
				err := r.Handle(conn)
				if err != nil && !errors.Is(err, context.Canceled) {
					r.l.Error(err)
				}
			}()
		}
	}
}
