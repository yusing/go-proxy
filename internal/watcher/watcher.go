package watcher

import (
	"context"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type Event = events.Event

type Watcher interface {
	Events(ctx context.Context) (<-chan Event, <-chan E.NestedError)
}
