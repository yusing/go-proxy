package watcher

import (
	"context"

	E "github.com/yusing/go-proxy/error"
	"github.com/yusing/go-proxy/watcher/events"
)

type Event = events.Event

type Watcher interface {
	Events(ctx context.Context) (<-chan Event, <-chan E.NestedError)
}
