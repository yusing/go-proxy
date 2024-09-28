package watcher

import (
	"context"

	E "github.com/yusing/go-proxy/internal/error"
)

type fileWatcher struct {
	relPath string
	eventCh chan Event
	errCh   chan E.NestedError
}

func (fw *fileWatcher) Events(ctx context.Context) (<-chan Event, <-chan E.NestedError) {
	return fw.eventCh, fw.errCh
}
