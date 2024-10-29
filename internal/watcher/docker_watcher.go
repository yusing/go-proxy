package watcher

import (
	"context"
	"time"

	docker_events "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/rs/zerolog"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type (
	DockerWatcher struct {
		zerolog.Logger

		host        string
		client      D.Client
		clientOwned bool
	}
	DockerListOptions = docker_events.ListOptions
)

// https://docs.docker.com/reference/api/engine/version/v1.47/#tag/System/operation/SystemPingHead
var (
	DockerFilterContainer = filters.Arg("type", string(docker_events.ContainerEventType))
	DockerFilterStart     = filters.Arg("event", string(docker_events.ActionStart))
	DockerFilterStop      = filters.Arg("event", string(docker_events.ActionStop))
	DockerFilterDie       = filters.Arg("event", string(docker_events.ActionDie))
	DockerFilterDestroy   = filters.Arg("event", string(docker_events.ActionDestroy))
	DockerFilterKill      = filters.Arg("event", string(docker_events.ActionKill))
	DockerFilterPause     = filters.Arg("event", string(docker_events.ActionPause))
	DockerFilterUnpause   = filters.Arg("event", string(docker_events.ActionUnPause))

	NewDockerFilter = filters.NewArgs

	optionsDefault = DockerListOptions{Filters: NewDockerFilter(
		DockerFilterContainer,
		DockerFilterStart,
		// DockerFilterStop,
		DockerFilterDie,
		DockerFilterDestroy,
	)}

	dockerWatcherRetryInterval = 3 * time.Second
)

func DockerFilterContainerNameID(nameOrID string) filters.KeyValuePair {
	return filters.Arg("container", nameOrID)
}

func NewDockerWatcher(host string) DockerWatcher {
	return DockerWatcher{
		host:        host,
		clientOwned: true,
		Logger: logger.With().
			Str("type", "docker").
			Str("host", host).
			Logger(),
	}
}

func NewDockerWatcherWithClient(client D.Client) DockerWatcher {
	return DockerWatcher{
		client: client,
		Logger: logger.With().
			Str("type", "docker").
			Str("host", client.DaemonHost()).
			Logger(),
	}
}

func (w DockerWatcher) Events(ctx context.Context) (<-chan Event, <-chan E.Error) {
	return w.EventsWithOptions(ctx, optionsDefault)
}

func (w DockerWatcher) EventsWithOptions(ctx context.Context, options DockerListOptions) (<-chan Event, <-chan E.Error) {
	eventCh := make(chan Event)
	errCh := make(chan E.Error)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		defer func() {
			if w.clientOwned && w.client.Connected() {
				w.client.Close()
			}
		}()

		if !w.client.Connected() {
			var err error
			attempts := 0
			for {
				w.client, err = D.ConnectClient(w.host)
				if err == nil {
					break
				}
				attempts++
				errCh <- E.Errorf("docker connection attempt #%d: %w", attempts, err)
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(dockerWatcherRetryInterval)
				}
			}
		}

		defer w.client.Close()

		cEventCh, cErrCh := w.client.Events(ctx, options)

		for {
			select {
			case <-ctx.Done():
				if err := E.From(ctx.Err()); err != nil && !err.Is(context.Canceled) {
					errCh <- err
				}
				return
			case msg := <-cEventCh:
				action, ok := events.DockerEventMap[msg.Action]
				if !ok {
					w.Debug().Msgf("ignored unknown docker event: %s for container %s", msg.Action, msg.Actor.Attributes["name"])
					continue
				}
				event := Event{
					Type:            events.EventTypeDocker,
					ActorID:         msg.Actor.ID,
					ActorAttributes: msg.Actor.Attributes, // labels
					ActorName:       msg.Actor.Attributes["name"],
					Action:          action,
				}
				eventCh <- event
			case err := <-cErrCh:
				if err == nil {
					continue
				}
				errCh <- E.From(err)
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(dockerWatcherRetryInterval)
					cEventCh, cErrCh = w.client.Events(ctx, options)
				}
			}
		}
	}()

	return eventCh, errCh
}
