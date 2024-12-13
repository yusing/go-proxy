package route

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher"
	E "github.com/yusing/go-proxy/internal/error"
	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/route/entry"
	"github.com/yusing/go-proxy/internal/route/routes"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"
)

type StreamRoute struct {
	*entry.StreamEntry

	net.Stream

	HealthMon health.HealthMonitor `json:"health"`

	task task.Task

	l zerolog.Logger
}

func NewStreamRoute(entry *entry.StreamEntry) (impl, E.Error) {
	// TODO: support non-coherent scheme
	if !entry.Scheme.IsCoherent() {
		return nil, E.Errorf("unsupported scheme: %v -> %v", entry.Scheme.ListeningScheme, entry.Scheme.ProxyScheme)
	}
	return &StreamRoute{
		StreamEntry: entry,
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

	r.task = providerSubtask
	r.Stream = NewStream(r)

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
		if entry.IsDocker(r) {
			client, err := docker.ConnectClient(r.Idlewatcher.DockerHost)
			if err == nil {
				fallback := monitor.NewRawHealthChecker(r.TargetURL(), r.HealthCheck)
				r.HealthMon = monitor.NewDockerHealthMonitor(client, r.Idlewatcher.ContainerID, r.HealthCheck, fallback)
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

	r.task.OnFinished("close stream", func() {
		if err := r.Stream.Close(); err != nil {
			E.LogError("close stream failed", err, &r.l)
		}
	})

	r.l.Info().
		Int("port", int(r.Port.ListeningPort)).
		Msg("listening")

	if r.HealthMon != nil {
		healthMonTask := r.task.Subtask("health monitor")
		if err := r.HealthMon.Start(healthMonTask); err != nil {
			E.LogWarn("health monitor error", err, &r.l)
			healthMonTask.Finish(err)
		}
	}

	go r.acceptConnections()

	routes.SetStreamRoute(string(r.Alias), r)
	r.task.OnFinished("remove from route table", func() {
		routes.DeleteStreamRoute(string(r.Alias))
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
			connTask := r.task.Subtask("connection")
			go func() {
				err := r.Stream.Handle(conn)
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
