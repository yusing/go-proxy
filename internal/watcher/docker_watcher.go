package watcher

import (
	"context"
	"errors"
	"time"

	docker_events "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/gperr"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type (
	DockerWatcher struct {
		host        string
		client      *docker.SharedClient
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

	reloadTrigger = Event{
		Type:            events.EventTypeDocker,
		Action:          events.ActionForceReload,
		ActorAttributes: map[string]string{},
		ActorName:       "",
		ActorID:         "",
	}
)

func DockerFilterContainerNameID(nameOrID string) filters.KeyValuePair {
	return filters.Arg("container", nameOrID)
}

func NewDockerWatcher(host string) *DockerWatcher {
	return &DockerWatcher{
		host:        host,
		clientOwned: true,
	}
}

func NewDockerWatcherWithClient(client *docker.SharedClient) *DockerWatcher {
	return &DockerWatcher{
		client: client,
	}
}

func (w *DockerWatcher) Events(ctx context.Context) (<-chan Event, <-chan gperr.Error) {
	return w.EventsWithOptions(ctx, optionsDefault)
}

func (w *DockerWatcher) Close() {
	if w.clientOwned && w.client.Connected() {
		w.client.Close()
	}
}

func (w *DockerWatcher) parseError(err error) gperr.Error {
	if errors.Is(err, context.DeadlineExceeded) {
		return gperr.New("docker client connection timeout")
	}
	if client.IsErrConnectionFailed(err) {
		return gperr.New("docker client connection failure")
	}
	return gperr.Wrap(err)
}

func (w *DockerWatcher) checkConnection(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, dockerWatcherRetryInterval)
	defer cancel()
	_, err := w.client.Ping(ctx)
	if err != nil {
		return false
	}
	return true
}

func (w *DockerWatcher) EventsWithOptions(ctx context.Context, options DockerListOptions) (<-chan Event, <-chan gperr.Error) {
	eventCh := make(chan Event)
	errCh := make(chan gperr.Error)

	go func() {
		defer func() {
			close(eventCh)
			close(errCh)
			w.Close()
		}()

		if !w.client.Connected() {
			var err error
			w.client, err = docker.ConnectClient(w.host)
			attempts := 0
			retryTicker := time.NewTicker(dockerWatcherRetryInterval)
			for err != nil {
				attempts++
				errCh <- gperr.Errorf("docker connection attempt #%d: %w", attempts, err)
				select {
				case <-ctx.Done():
					retryTicker.Stop()
					return
				case <-retryTicker.C:
					w.client, err = docker.ConnectClient(w.host)
				}
			}
			retryTicker.Stop()
		}

		defer w.Close()

		cEventCh, cErrCh := w.client.Events(ctx, options)
		defer logging.Debug().Str("host", w.host).Msg("docker watcher closed")
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-cEventCh:
				action, ok := events.DockerEventMap[msg.Action]
				if !ok {
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
				errCh <- w.parseError(err)
				// trigger reload (clear routes)
				eventCh <- reloadTrigger
				for !w.checkConnection(ctx) {
					select {
					case <-ctx.Done():
						return
					case <-time.After(dockerWatcherRetryInterval):
						continue
					}
				}
				// connection successful, trigger reload (reload routes)
				eventCh <- reloadTrigger
				// reopen event channel
				cEventCh, cErrCh = w.client.Events(ctx, options)
			}
		}
	}()

	return eventCh, errCh
}
