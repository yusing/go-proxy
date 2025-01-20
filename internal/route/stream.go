package route

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/route/entry"
	"github.com/yusing/go-proxy/internal/route/routes"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"
)

// TODO: support stream load balance.
type StreamRoute struct {
	*entry.StreamEntry

	net.Stream `json:"-"`

	HealthMon health.HealthMonitor `json:"health"`

	task *task.Task

	l zerolog.Logger
}

func NewStreamRoute(entry *entry.StreamEntry) (impl, E.Error) {
	// TODO: support non-coherent scheme
	if !entry.Scheme.IsCoherent() {
		return nil, E.Errorf("unsupported scheme: %v -> %v", entry.Scheme.ListeningScheme, entry.Scheme.ProxyScheme)
	}
	return &StreamRoute{
		StreamEntry: entry,
		l: logging.With().
			Str("type", string(entry.Scheme.ListeningScheme)).
			Str("name", entry.TargetName()).
			Logger(),
	}, nil
}

func (r *StreamRoute) String() string {
	return "stream " + r.TargetName()
}

// Start implements task.TaskStarter.
func (r *StreamRoute) Start(parent task.Parent) E.Error {
	if entry.ShouldNotServe(r) {
		return nil
	}

	r.task = parent.Subtask("stream." + r.TargetName())
	r.Stream = NewStream(r)
	parent.OnCancel("finish", func() {
		r.task.Finish(nil)
	})

	switch {
	case entry.UseIdleWatcher(r):
		waker, err := idlewatcher.NewStreamWaker(parent, r.StreamEntry, r.Stream)
		if err != nil {
			r.task.Finish(err)
			return err
		}
		r.Stream = waker
		r.HealthMon = waker
	case entry.UseHealthCheck(r):
		if entry.IsDocker(r) {
			client, err := docker.ConnectClient(r.Idlewatcher.DockerHost)
			if err == nil {
				fallback := monitor.NewRawHealthChecker(r.TargetURL(), r.Raw.HealthCheck)
				r.HealthMon = monitor.NewDockerHealthMonitor(client, r.Idlewatcher.ContainerID, r.TargetName(), r.Raw.HealthCheck, fallback)
				r.task.OnCancel("close_docker_client", client.Close)
			}
		}
		if r.HealthMon == nil {
			r.HealthMon = monitor.NewRawHealthMonitor(r.TargetURL(), r.Raw.HealthCheck)
		}
	}

	if err := r.Stream.Setup(); err != nil {
		r.task.Finish(err)
		return E.From(err)
	}

	r.l.Info().
		Int("port", int(r.Port.ListeningPort)).
		Msg("listening")

	if r.HealthMon != nil {
		if err := r.HealthMon.Start(r.task); err != nil {
			E.LogWarn("health monitor error", err, &r.l)
		}
	}

	go r.acceptConnections()

	routes.SetStreamRoute(r.TargetName(), r)
	r.task.OnCancel("entrypoint_remove_route", func() {
		routes.DeleteStreamRoute(r.TargetName())
	})
	return nil
}

// Task implements task.TaskStarter.
func (r *StreamRoute) Task() *task.Task {
	return r.task
}

// Finish implements task.TaskFinisher.
func (r *StreamRoute) Finish(reason any) {
	r.task.Finish(reason)
}

func (r *StreamRoute) HealthMonitor() health.HealthMonitor {
	return r.HealthMon
}

func (r *StreamRoute) acceptConnections() {
	defer r.task.Finish("listener closed")

	for {
		select {
		case <-r.task.Context().Done():
			return
		default:
			conn, err := r.Stream.Accept()
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
			go func() {
				err := r.Stream.Handle(conn)
				if err != nil && !errors.Is(err, context.Canceled) {
					E.LogError("handle connection error", err, &r.l)
				}
			}()
		}
	}
}
