package route

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
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

	stream net.Stream `json:"-"`

	HealthMon health.HealthMonitor `json:"health"`

	task task.Task

	l zerolog.Logger
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
		return nil, E.Errorf("unsupported scheme: %v -> %v", entry.Scheme.ListeningScheme, entry.Scheme.ProxyScheme)
	}
	return &StreamRoute{
		StreamEntry: entry,
		task:        task.DummyTask(),
		l: logger.With().
			Str("type", string(entry.Scheme.ListeningScheme)).
			Str("name", entry.TargetName()).
			Logger(),
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
		r.l.Error().Msg("healthCheck.disabled cannot be false when loadbalancer or idlewatcher is enabled")
		r.HealthCheck.Disable = false
	}

	r.task = providerSubtask
	r.stream = NewStream(r)

	switch {
	case entry.UseIdleWatcher(r):
		wakerTask := providerSubtask.Parent().Subtask("waker for " + string(r.Alias))
		waker, err := idlewatcher.NewStreamWaker(wakerTask, r.StreamEntry, r.stream)
		if err != nil {
			r.task.Finish(err)
			return err
		}
		r.stream = waker
		r.HealthMon = waker
	case entry.UseHealthCheck(r):
		r.HealthMon = health.NewRawHealthMonitor(r.TargetURL(), r.HealthCheck)
	}

	if err := r.stream.Setup(); err != nil {
		r.task.Finish(err)
		return E.From(err)
	}

	r.task.OnFinished("close stream", func() {
		if err := r.stream.Close(); err != nil {
			E.LogError("close stream failed", err, &r.l)
		}
	})

	r.l.Info().
		Int("port", int(r.Port.ListeningPort)).
		Msg("listening")

	if r.HealthMon != nil {
		if err := r.HealthMon.Start(r.task.Subtask("health monitor")); err != nil {
			E.LogWarn("health monitor error", err, &r.l)
		}
	}

	go r.acceptConnections()
	streamRoutes.Store(string(r.Alias), r)
	r.task.OnFinished("remove from route table", func() {
		streamRoutes.Delete(string(r.Alias))
	})
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
			conn, err := r.stream.Accept()
			if err != nil {
				select {
				case <-r.task.Context().Done():
				default:
					E.LogError("accept connection error", err, &r.l)
				}
				r.task.Finish(err)
				return
			}
			if conn == nil {
				panic("connection is nil")
			}
			connTask := r.task.Subtask("connection")
			go func() {
				err := r.stream.Handle(conn)
				if err != nil && !errors.Is(err, context.Canceled) {
					E.LogError("handle connection error", err, &r.l)
					connTask.Finish(err)
				} else {
					connTask.Finish("closed")
				}
			}()
		}
	}
}
