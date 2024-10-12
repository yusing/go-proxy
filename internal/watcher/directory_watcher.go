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

type DirWatcher struct {
	dir string
	w   *fsnotify.Watcher

	fwMap F.Map[string, *fileWatcher]
	mu    sync.Mutex

	eventCh chan Event
	errCh   chan E.NestedError

	ctx context.Context
}

// NewDirectoryWatcher returns a DirWatcher instance.
//
// The DirWatcher watches the given directory for file system events.
// Currently, only events on files directly in the given directory are watched, not
// recursively.
//
// Note that the returned DirWatcher is not ready to use until the goroutine
// started by NewDirectoryWatcher has finished.
func NewDirectoryWatcher(ctx context.Context, dirPath string) *DirWatcher {
	//! subdirectories are not watched
	w, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Panicf("unable to create fs watcher: %s", err)
	}
	if err = w.Add(dirPath); err != nil {
		logrus.Panicf("unable to create fs watcher: %s", err)
	}
	helper := &DirWatcher{
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

func (h *DirWatcher) Events(_ context.Context) (<-chan Event, <-chan E.NestedError) {
	return h.eventCh, h.errCh
}

func (h *DirWatcher) Add(relPath string) Watcher {
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
		<-h.ctx.Done()
		logrus.Debugf("file watcher %s stopped", relPath)
	}()
	h.fwMap.Store(relPath, s)
	return s
}

func (h *DirWatcher) start() {
	defer close(h.eventCh)
	defer h.w.Close()
	defer logrus.Debugf("directory watcher %s stopped", h.dir)

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
				logrus.Debugf("sent event to directory watcher %s", h.dir)
			default:
				logrus.Debugf("failed to send event to directory watcher %s", h.dir)
			}

			// send event to file watcher too
			w, ok := h.fwMap.Load(relPath)
			if ok {
				select {
				case w.eventCh <- msg:
					logrus.Debugf("sent event to file watcher %s", relPath)
				default:
					logrus.Debugf("failed to send event to file watcher %s", relPath)
				}
			} else {
				logrus.Debugf("file watcher not found: %s", relPath)
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
