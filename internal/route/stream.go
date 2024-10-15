package route

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	url "github.com/yusing/go-proxy/internal/net/types"
	P "github.com/yusing/go-proxy/internal/proxy"
	PT "github.com/yusing/go-proxy/internal/proxy/fields"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type StreamRoute struct {
	*P.StreamEntry
	StreamImpl `json:"-"`

	HealthMon health.HealthMonitor `json:"health"`

	url url.URL

	task   common.Task
	cancel context.CancelFunc
	done   chan struct{}

	l logrus.FieldLogger

	mu sync.Mutex
}

type StreamImpl interface {
	Setup() error
	Accept() (any, error)
	Handle(conn any) error
	CloseListeners()
	String() string
}

var streamRoutes = F.NewMapOf[string, *StreamRoute]()

func GetStreamProxies() F.Map[string, *StreamRoute] {
	return streamRoutes
}

func NewStreamRoute(entry *P.StreamEntry) (*StreamRoute, E.NestedError) {
	// TODO: support non-coherent scheme
	if !entry.Scheme.IsCoherent() {
		return nil, E.Unsupported("scheme", fmt.Sprintf("%v -> %v", entry.Scheme.ListeningScheme, entry.Scheme.ProxyScheme))
	}
	url, err := url.ParseURL(fmt.Sprintf("%s://%s:%d", entry.Scheme.ProxyScheme, entry.Host, entry.Port.ProxyPort))
	if err != nil {
		// !! should not happen
		panic(err)
	}
	base := &StreamRoute{
		StreamEntry: entry,
		url:         url,
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
	return fmt.Sprintf("stream %s", r.Alias)
}

func (r *StreamRoute) URL() url.URL {
	return r.url
}

func (r *StreamRoute) Start() E.NestedError {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Port.ProxyPort == PT.NoPort || r.task != nil {
		return nil
	}
	r.task, r.cancel = common.NewTaskWithCancel(r.String())
	if err := r.Setup(); err != nil {
		return E.FailWith("setup", err)
	}
	r.done = make(chan struct{})
	r.l.Infof("listening on port %d", r.Port.ListeningPort)
	go r.acceptConnections()
	if !r.Healthcheck.Disabled {
		r.HealthMon = health.NewRawHealthMonitor(r.task, r.URL(), r.Healthcheck)
		r.HealthMon.Start()
	}
	streamRoutes.Store(string(r.Alias), r)
	return nil
}

func (r *StreamRoute) Stop() E.NestedError {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.task == nil {
		return nil
	}

	streamRoutes.Delete(string(r.Alias))

	if r.HealthMon != nil {
		r.HealthMon.Stop()
		r.HealthMon = nil
	}

	r.cancel()
	r.CloseListeners()

	<-r.done
	return nil
}

func (r *StreamRoute) Started() bool {
	return r.task != nil
}

func (r *StreamRoute) acceptConnections() {
	var connWg sync.WaitGroup

	task := r.task.Subtask("%s accept connections", r.String())

	defer func() {
		connWg.Wait()
		task.Finished()
		r.task.Finished()
		r.task, r.cancel = nil, nil
		close(r.done)
		r.done = nil
	}()

	for {
		select {
		case <-task.Context().Done():
			return
		default:
			conn, err := r.Accept()
			if err != nil {
				select {
				case <-task.Context().Done():
					return
				default:
					var nErr *net.OpError
					ok := errors.As(err, &nErr)
					if !(ok && nErr.Timeout()) {
						r.l.Error(err)
					}
					continue
				}
			}
			connWg.Add(1)
			go func() {
				err := r.Handle(conn)
				if err != nil && !errors.Is(err, context.Canceled) {
					r.l.Error(err)
				}
				connWg.Done()
			}()
		}
	}
}
