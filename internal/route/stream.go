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
	"github.com/yusing/go-proxy/internal/route/routes"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"
)

// TODO: support stream load balance.
type StreamRoute struct {
	*Route

	net.Stream `json:"-"`

	HealthMon health.HealthMonitor `json:"health"`

	task *task.Task

	l zerolog.Logger
}

func NewStreamRoute(base *Route) (route.Route, E.Error) {
	// TODO: support non-coherent scheme
	return &StreamRoute{
		Route: base,
		l: logging.With().
			Str("type", string(base.Scheme)).
			Str("name", base.TargetName()).
			Logger(),
	}, nil
}

func (r *StreamRoute) String() string {
	return "stream " + r.TargetName()
}

// Start implements task.TaskStarter.
func (r *StreamRoute) Start(parent task.Parent) E.Error {
	r.task = parent.Subtask("stream." + r.TargetName())
	r.Stream = NewStream(r)
	parent.OnCancel("finish", func() {
		r.task.Finish(nil)
	})

	switch {
	case r.UseIdleWatcher():
		waker, err := idlewatcher.NewStreamWaker(parent, r, r.Stream)
		if err != nil {
			r.task.Finish(err)
			return err
		}
		r.Stream = waker
		r.HealthMon = waker
	case r.UseHealthCheck():
		if r.IsDocker() {
			client, err := docker.ConnectClient(r.IdlewatcherConfig().DockerHost)
			if err == nil {
				fallback := monitor.NewRawHealthChecker(r.TargetURL(), r.HealthCheck)
				r.HealthMon = monitor.NewDockerHealthMonitor(client, r.IdlewatcherConfig().ContainerID, r.TargetName(), r.HealthCheck, fallback)
				r.task.OnCancel("close_docker_client", client.Close)
			}
		}
		if r.HealthMon == nil {
			r.HealthMon = monitor.NewRawHealthMonitor(r.TargetURL(), r.HealthCheck)
		}
	}

	if err := r.Stream.Setup(); err != nil {
		r.task.Finish(err)
		return E.From(err)
	}

	r.l.Info().Int("port", r.Port.Listening).Msg("listening")

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
