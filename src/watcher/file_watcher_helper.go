package watcher

import (
	"context"
	"errors"
	"path"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
)

type fileWatcherHelper struct {
	w  *fsnotify.Watcher
	m  map[string]*fileWatcherStream
	wg sync.WaitGroup
	mu sync.Mutex
}

type fileWatcherStream struct {
	*fileWatcher
	stopped chan struct{}
	eventCh chan Event
	errCh   chan E.NestedError
}

func newFileWatcherHelper() *fileWatcherHelper {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Panicf("unable to create fs watcher: %s", err)
	}
	// watch config path for all changes
	err = w.Add(common.ConfigBasePath)
	if err != nil {
		logrus.Panicf("unable to create fs watcher: %s", err)
	}
	helper := &fileWatcherHelper{
		w: w,
		m: make(map[string]*fileWatcherStream),
	}
	go helper.start()
	return helper
}

func (h *fileWatcherHelper) Add(ctx context.Context, w *fileWatcher) (<-chan Event, <-chan E.NestedError) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// check if the watcher already exists
	s, ok := h.m[w.filename]
	if ok {
		return s.eventCh, s.errCh
	}
	s = &fileWatcherStream{
		fileWatcher: w,
		stopped:     make(chan struct{}),
		eventCh:     make(chan Event),
		errCh:       make(chan E.NestedError),
	}
	go func() {
		select {
		case <-ctx.Done():
			h.Remove(w)
			return
		case <-s.stopped:
			return
		}
	}()
	h.m[w.filename] = s
	return s.eventCh, s.errCh
}

func (h *fileWatcherHelper) Remove(w *fileWatcher) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.m[w.filename].stopped <- struct{}{}
	delete(h.m, w.filename)
}

// deinit closes the fs watcher
// and waits for the start() loop to finish
func (h *fileWatcherHelper) close() {
	_ = h.w.Close()
	h.wg.Wait() // wait for `start()` loop to finish
}

func (h *fileWatcherHelper) start() {
	defer h.wg.Done()

	for {
		select {
		case event, ok := <-h.w.Events:
			if !ok {
				// closed manually?
				fsLogger.Error("channel closed")
				return
			}
			// retrieve the watcher
			w, ok := h.m[path.Base(event.Name)]
			if !ok {
				// watcher for this file does not exist
				continue
			}

			msg := Event{ActorName: w.filename}
			switch {
			case event.Has(fsnotify.Create):
				msg.Action = ActionCreated
			case event.Has(fsnotify.Write):
				msg.Action = ActionModified
			case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
				msg.Action = ActionDeleted
			default: // ignore other events
				continue
			}

			// send event
			w.eventCh <- msg
		case err := <-h.w.Errors:
			if errors.Is(err, fsnotify.ErrClosed) {
				// closed manually?
				return
			}
			fsLogger.Error(err)
		}
	}
}

var fsLogger = logrus.WithField("module", "fsnotify")
