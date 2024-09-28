package watcher

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/internal/error"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type dirWatcher struct {
	dir string
	w   *fsnotify.Watcher

	fwMap F.Map[string, *fileWatcher]
	mu    sync.Mutex

	eventCh chan Event
	errCh   chan E.NestedError

	ctx context.Context
}

func NewDirectoryWatcher(ctx context.Context, dirPath string) *dirWatcher {
	//! subdirectories are not watched
	w, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Panicf("unable to create fs watcher: %s", err)
	}
	if err = w.Add(dirPath); err != nil {
		logrus.Panicf("unable to create fs watcher: %s", err)
	}
	helper := &dirWatcher{
		dir:     dirPath,
		w:       w,
		fwMap:   F.NewMapOf[string, *fileWatcher](),
		eventCh: make(chan Event),
		errCh:   make(chan E.NestedError),
		ctx:     ctx,
	}
	go helper.start()
	return helper
}

func (h *dirWatcher) Events(_ context.Context) (<-chan Event, <-chan E.NestedError) {
	return h.eventCh, h.errCh
}

func (h *dirWatcher) Add(relPath string) *fileWatcher {
	h.mu.Lock()
	defer h.mu.Unlock()

	// check if the watcher already exists
	s, ok := h.fwMap.Load(relPath)
	if ok {
		return s
	}
	s = &fileWatcher{
		relPath: relPath,
		eventCh: make(chan Event),
		errCh:   make(chan E.NestedError),
	}
	go func() {
		defer func() {
			close(s.eventCh)
			close(s.errCh)
		}()
		for {
			select {
			case <-h.ctx.Done():
				return
			case _, ok := <-h.eventCh:
				if !ok { // directory watcher closed
					return
				}
			}
		}
	}()
	h.fwMap.Store(relPath, s)
	return s
}

func (h *dirWatcher) start() {
	defer close(h.eventCh)
	defer h.w.Close()

	for {
		select {
		case <-h.ctx.Done():
			return
		case fsEvent, ok := <-h.w.Events:
			if !ok {
				return
			}
			// retrieve the watcher
			relPath := strings.TrimPrefix(fsEvent.Name, h.dir)
			relPath = strings.TrimPrefix(relPath, "/")

			msg := Event{
				Type:      events.EventTypeFile,
				ActorName: relPath,
			}
			switch {
			case fsEvent.Has(fsnotify.Write):
				msg.Action = events.ActionFileWritten
			case fsEvent.Has(fsnotify.Create):
				msg.Action = events.ActionFileCreated
			case fsEvent.Has(fsnotify.Remove):
				msg.Action = events.ActionFileDeleted
			case fsEvent.Has(fsnotify.Rename):
				msg.Action = events.ActionFileRenamed
			default: // ignore other events
				continue
			}

			// send event to directory watcher
			select {
			case h.eventCh <- msg:
			default:
			}

			// send event to file watcher too
			w, ok := h.fwMap.Load(relPath)
			if ok {
				select {
				case w.eventCh <- msg:
				default:
				}
			}
		case err := <-h.w.Errors:
			if errors.Is(err, fsnotify.ErrClosed) {
				// closed manually?
				return
			}
			select {
			case h.errCh <- E.From(err):
			default:
			}
		}
	}
}
