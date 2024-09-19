package watcher

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	. "github.com/yusing/go-proxy/watcher/event"
)

type DockerWatcher struct {
	host string
}

func NewDockerWatcher(host string) *DockerWatcher {
	return &DockerWatcher{host: host}
}

func (w *DockerWatcher) Events(ctx context.Context) (<-chan Event, <-chan E.NestedError) {
	eventCh := make(chan Event)
	errCh := make(chan E.NestedError)
	started := make(chan struct{})

	go func() {
		defer close(errCh)

		var cl D.Client
		var err E.NestedError
		for range 3 {
			cl, err = D.ConnectClient(w.host)
			if err.NoError() {
				break
			}
			errCh <- err
			time.Sleep(1 * time.Second)
		}
		if err.HasError() {
			errCh <- E.Failure("connecting to docker")
			return
		}
		defer cl.Close()

		cEventCh, cErrCh := cl.Events(ctx, dwOptions)
		started <- struct{}{}

		for {
			select {
			case <-ctx.Done():
				if err := <-cErrCh; err != nil {
					errCh <- E.From(err)
				}
				return
			case msg := <-cEventCh:
				var Action Action
				switch msg.Action {
				case events.ActionStart:
					Action = ActionCreated
				case events.ActionDie:
					Action = ActionStopped
				default: // NOTE: should not happen
					Action = ActionModified
				}
				eventCh <- Event{
					Type:            EventTypeDocker,
					ActorID:         msg.Actor.ID,
					ActorAttributes: msg.Actor.Attributes, // labels
					ActorName:       msg.Actor.Attributes["name"],
					Action:          Action,
				}
			case err := <-cErrCh:
				if err == nil {
					continue
				}
				errCh <- E.From(err)
				select {
				case <-ctx.Done():
					return
				default:
					if D.IsErrConnectionFailed(err) {
						time.Sleep(100 * time.Millisecond)
						cEventCh, cErrCh = cl.Events(ctx, dwOptions)
					}
				}
			}
		}
	}()
	<-started

	return eventCh, errCh
}

var dwOptions = events.ListOptions{Filters: filters.NewArgs(
	filters.Arg("type", string(events.ContainerEventType)),
	filters.Arg("event", string(events.ActionStart)),
	filters.Arg("event", string(events.ActionDie)), // 'stop' already triggering 'die'
)}
