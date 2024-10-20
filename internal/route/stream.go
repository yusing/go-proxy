package route

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher"
	E "github.com/yusing/go-proxy/internal/error"
	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	"github.com/yusing/go-proxy/internal/task"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type StreamRoute struct {
	*entry.StreamEntry
	net.Stream `json:"-"`

	HealthMon health.HealthMonitor `json:"health"`

	task task.Task

	l logrus.FieldLogger
}

var (
	streamRoutes   = F.NewMapOf[string, *StreamRoute]()
	streamRoutesMu sync.Mutex
)

func GetStreamProxies() F.Map[string, *StreamRoute] {
	return streamRoutes
}

func NewStreamRoute(entry *entry.StreamEntry) (impl, E.Error) {
	// TODO: support non-coherent scheme
	if !entry.Scheme.IsCoherent() {
		return nil, E.Unsupported("scheme", fmt.Sprintf("%v -> %v", entry.Scheme.ListeningScheme, entry.Scheme.ProxyScheme))
	}
	return &StreamRoute{
		StreamEntry: entry,
		task:        task.DummyTask(),
	}, nil
}

func (r *StreamRoute) String() string {
	return fmt.Sprintf("stream %s", r.Alias)
}

// Start implements task.TaskStarter.
func (r *StreamRoute) Start(providerSubtask task.Task) E.Error {
	if entry.ShouldNotServe(r) {
		providerSubtask.Finish("should not serve")
		return nil
	}

	streamRoutesMu.Lock()
	defer streamRoutesMu.Unlock()

	if r.HealthCheck.Disable && (entry.UseLoadBalance(r) || entry.UseIdleWatcher(r)) {
		logrus.Warnf("%s.healthCheck.disabled cannot be false when loadbalancer or idlewatcher is enabled", r.Alias)
		r.HealthCheck.Disable = true
	}

	// if r.Scheme.ListeningScheme.IsTCP() {
	// 	r.Stream = NewTCPRoute(r)
	// } else {
	// 	r.Stream = NewUDPRoute(r)
	// }
	r.task = providerSubtask
	r.Stream = NewRawStreamRoute(r)
	r.l = logrus.WithField("route", r.Stream.String())

	switch {
	case entry.UseIdleWatcher(r):
		wakerTask := providerSubtask.Parent().Subtask("waker for " + string(r.Alias))
		waker, err := idlewatcher.NewStreamWaker(wakerTask, r.StreamEntry, r.Stream)
		if err != nil {
			r.task.Finish(err)
			return err
		}
		r.Stream = waker
		r.HealthMon = waker
	case entry.UseHealthCheck(r):
		r.HealthMon = health.NewRawHealthMonitor(r.TargetURL(), r.HealthCheck)
	}

	if err := r.Setup(); err != nil {
		r.task.Finish(err)
		return E.FailWith("setup", err)
	}

	r.task.OnFinished("close stream", func() {
		if err := r.Close(); err != nil {
			r.l.Error("close stream error: ", err)
		}
	})
	r.task.OnFinished("remove from route table", func() {
		streamRoutes.Delete(string(r.Alias))
	})

	r.l.Infof("listening on %s port %d", r.Scheme.ListeningScheme, r.Port.ListeningPort)

	if r.HealthMon != nil {
		if err := r.HealthMon.Start(r.task.Subtask("health monitor")); err != nil {
			logrus.Warn("health monitor error: ", err)
		}
	}

	go r.acceptConnections()
	streamRoutes.Store(string(r.Alias), r)
	return nil
}

func (r *StreamRoute) Finish(reason any) {
	r.task.Finish(reason)
}

func (r *StreamRoute) acceptConnections() {
	defer r.task.Finish("listener closed")

	for {
		select {
		case <-r.task.Context().Done():
			return
		default:
			conn, err := r.Accept()
			if err != nil {
				select {
				case <-r.task.Context().Done():
				default:
					r.l.Error("accept connection error: ", err)
					r.task.Finish(err)
				}
				return
			}
			connTask := r.task.Subtask(fmt.Sprintf("connection from %s", conn.RemoteAddr()))
			go func() {
				err := r.Handle(conn)
				if err != nil && !errors.Is(err, context.Canceled) {
					r.l.Error(err)
				} else {
					connTask.Finish("connection closed")
				}
				conn.Close()
			}()
		}
	}
}
