package watcher

import (
	"context"
	"fmt"
	"time"

	docker_events "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/sirupsen/logrus"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type (
	DockerWatcher struct {
		host   string
		client D.Client
		logrus.FieldLogger
	}
	DockerListOptions = docker_events.ListOptions
)

// https://docs.docker.com/reference/api/engine/version/v1.47/#tag/System/operation/SystemPingHead
var (
	DockerFilterContainer = filters.Arg("type", string(docker_events.ContainerEventType))
	DockerFilterStart     = filters.Arg("event", string(docker_events.ActionStart))
	DockerFilterStop      = filters.Arg("event", string(docker_events.ActionStop))
	DockerFilterDie       = filters.Arg("event", string(docker_events.ActionDie))
	DockerFilterKill      = filters.Arg("event", string(docker_events.ActionKill))
	DockerFilterPause     = filters.Arg("event", string(docker_events.ActionPause))
	DockerFilterUnpause   = filters.Arg("event", string(docker_events.ActionUnPause))

	NewDockerFilter = filters.NewArgs

	dockerWatcherRetryInterval = 3 * time.Second
)

func DockerrFilterContainerName(name string) filters.KeyValuePair {
	return filters.Arg("container", name)
}

func NewDockerWatcher(host string) DockerWatcher {
	return DockerWatcher{
		host: host,
		FieldLogger: (logrus.
			WithField("module", "docker_watcher").
			WithField("host", host))}
}

func NewDockerWatcherWithClient(client D.Client) DockerWatcher {
	return DockerWatcher{
		client: client,
		FieldLogger: (logrus.
			WithField("module", "docker_watcher").
			WithField("host", client.DaemonHost()))}
}

func (w DockerWatcher) Events(ctx context.Context) (<-chan Event, <-chan E.NestedError) {
	return w.EventsWithOptions(ctx, optionsWatchAll)
}

func (w DockerWatcher) EventsWithOptions(ctx context.Context, options DockerListOptions) (<-chan Event, <-chan E.NestedError) {
	eventCh := make(chan Event)
	errCh := make(chan E.NestedError)

	eventsCtx, eventsCancel := context.WithCancel(ctx)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		defer func() {
			if w.client.Connected() {
				w.client.Close()
			}
		}()

		if !w.client.Connected() {
			var err E.NestedError
			attempts := 0
			for {
				w.client, err = D.ConnectClient(w.host)
				if err == nil {
					break
				}
				attempts++
				errCh <- E.FailWith(fmt.Sprintf("docker connection attempt #%d", attempts), err)
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(dockerWatcherRetryInterval)
				}
			}
		}

		w.Debugf("client connected")

		cEventCh, cErrCh := w.client.Events(eventsCtx, options)

		w.Debugf("watcher started")

		for {
			select {
			case <-ctx.Done():
				if err := E.From(ctx.Err()); err != nil && err.IsNot(context.Canceled) {
					errCh <- err
				}
				return
			case msg := <-cEventCh:
				action, ok := events.DockerEventMap[msg.Action]
				if !ok {
					w.Debugf("ignored unknown docker event: %s for container %s", msg.Action, msg.Actor.Attributes["name"])
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
					eventsCancel()
					time.Sleep(dockerWatcherRetryInterval)
					eventsCtx, eventsCancel = context.WithCancel(ctx)
					cEventCh, cErrCh = w.client.Events(ctx, options)
				}
			}
		}
	}()

	return eventCh, errCh
}

var optionsWatchAll = DockerListOptions{Filters: NewDockerFilter(
	DockerFilterContainer,
	DockerFilterStart,
	DockerFilterStop,
	DockerFilterDie,
)}
