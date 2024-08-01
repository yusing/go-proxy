package watcher

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
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
			if err.IsNotNil() {
				break
			}
			errCh <- E.From(err)
			time.Sleep(1 * time.Second)
		}
		if err.IsNotNil() {
			errCh <- E.Failure("connect to docker")
			return
		}

		cEventCh, cErrCh := cl.Events(ctx, dwOptions)
		started <- struct{}{}

		for {
			select {
			case <-ctx.Done():
				errCh <- E.From(<-cErrCh)
				return
			case msg := <-cEventCh:
				containerName, ok := msg.Actor.Attributes["name"]
				if !ok {
					// NOTE: should not happen
					// but if it happens, just ignore it
					continue
				}
				eventCh <- Event{
					ActorName: containerName,
					Action:    ActionModified,
				}
			case err := <-cErrCh:
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

var dwOptions = types.EventsOptions{Filters: filters.NewArgs(
	filters.Arg("type", "container"),
	filters.Arg("event", "start"),
	filters.Arg("event", "die"), // 'stop' already triggering 'die'
)}
