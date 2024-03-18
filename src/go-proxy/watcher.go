package main

import (
	"path"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"

	"golang.org/x/net/context"
)

type Watcher interface {
	Start()
	Stop()
}

type watcherBase struct {
	name     string // for log / error output
	kind     string // for log / error output
	onChange func()
	l        logrus.FieldLogger
}

type fileWatcher struct {
	*watcherBase
	path     string
	onDelete func()
}

type dockerWatcher struct {
	*watcherBase
	client *client.Client
	stop   chan struct{}
	wg     sync.WaitGroup
}

func newWatcher(kind string, name string, onChange func()) *watcherBase {
	return &watcherBase{
		kind:     kind,
		name:     name,
		onChange: onChange,
		l:        wlog.WithFields(logrus.Fields{"kind": kind, "name": name}),
	}
}
func NewFileWatcher(p string, onChange func(), onDelete func()) Watcher {
	return &fileWatcher{
		watcherBase: newWatcher("File", path.Base(p), onChange),
		path:        p,
		onDelete:    onDelete,
	}
}

func NewDockerWatcher(c *client.Client, onChange func()) Watcher {
	return &dockerWatcher{
		watcherBase: newWatcher("Docker", c.DaemonHost(), onChange),
		client:      c,
		stop:        make(chan struct{}, 1),
	}
}

func (w *fileWatcher) Start() {
	if fsWatcher == nil {
		return
	}
	err := fsWatcher.Add(w.path)
	if err != nil {
		w.l.Error("failed to start: ", err)
	}
	fileWatchMap.Set(w.path, w)
}

func (w *fileWatcher) Stop() {
	fileWatchMap.Delete(w.path)
	err := fsWatcher.Remove(w.path)
	if err != nil {
		w.l.WithField("action", "stop").Error(err)
	}
}

func (w *dockerWatcher) Start() {
	dockerWatchMap.Set(w.name, w)
	w.wg.Add(1)
	go func() {
		w.watch()
		w.wg.Done()
	}()
}

func (w *dockerWatcher) Stop() {
	close(w.stop)
	w.stop = nil
	dockerWatchMap.Delete(w.name)
	w.wg.Wait()
}

func InitFSWatcher() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		wlog.Errorf("unable to create file watcher: %v", err)
		return
	}
	fsWatcher = w
	go watchFiles()
}

func InitDockerWatcher() {
	// stop all docker client on watcher stop
	go func() {
		<-dockerWatcherStop
		stopAllDockerClients()
	}()
}

func stopAllDockerClients() {
	ParallelForEachValue(
		dockerWatchMap.Iterator(),
		func(w *dockerWatcher) {
			w.Stop()
			err := w.client.Close()
			if err != nil {
				w.l.WithField("action", "stop").Error(err)
			}
			w.client = nil
		},
	)
}

func watchFiles() {
	defer fsWatcher.Close()
	for {
		select {
		case event, ok := <-fsWatcher.Events:
			if !ok {
				wlog.Error("file watcher channel closed")
				return
			}
			w, ok := fileWatchMap.UnsafeGet(event.Name)
			if !ok {
				wlog.Errorf("watcher for %s not found", event.Name)
			}
			switch {
			case event.Has(fsnotify.Write):
				w.l.Info("File change detected")
				w.onChange()
			case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
				w.l.Info("File renamed / deleted")
				w.onDelete()
			}
		case err := <-fsWatcher.Errors:
			wlog.Error(err)
		}
	}
}

func (w *dockerWatcher) watch() {
	filter := filters.NewArgs(
		filters.Arg("type", "container"),
		filters.Arg("event", "start"),
		filters.Arg("event", "die"), // 'stop' already triggering 'die'
	)
	listen := func() (<-chan events.Message, <-chan error) {
		return w.client.Events(context.Background(), types.EventsOptions{Filters: filter})
	}
	msgChan, errChan := listen()

	for {
		select {
		case <-w.stop:
			return
		case msg := <-msgChan:
			w.l.Info("container", msg.Actor.Attributes["name"], msg.Action)
			w.onChange()
		case err := <-errChan:
			w.l.Errorf("%s, retrying in 1s", err)
			time.Sleep(1 * time.Second)
			msgChan, errChan = listen()
		}
	}
}

var fsWatcher *fsnotify.Watcher
var (
	fileWatchMap   = NewSafeMap[string, *fileWatcher]()
	dockerWatchMap = NewSafeMap[string, *dockerWatcher]()
)
var (
	fsWatcherStop     = make(chan struct{}, 1)
	dockerWatcherStop = make(chan struct{}, 1)
)
