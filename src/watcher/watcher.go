package watcher

import (
	"context"

	E "github.com/yusing/go-proxy/error"
)

type Watcher interface {
	Events(ctx context.Context) (<-chan Event, <-chan E.NestedError)
}
