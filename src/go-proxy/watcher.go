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
	Dispose()
}

type watcherBase struct {
	name     string // for log / error output
	kind     string // for log / error output
	onChange func()
	l        logrus.FieldLogger
	sync.Mutex
}

type fileWatcher struct {
	*watcherBase
	path     string
	onDelete func()
}

type dockerWatcher struct {
	*watcherBase
	client *client.Client
	stopCh chan struct{}
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
		stopCh:      make(chan struct{}, 1),
	}
}

func (w *fileWatcher) Start() {
	w.Lock()
	defer w.Unlock()
	if fsWatcher == nil {
		return
	}
	err := fsWatcher.Add(w.path)
	if err != nil {
		w.l.Error("failed to start: ", err)
		return
	}
	fileWatchMap.Set(w.path, w)
}

func (w *fileWatcher) Stop() {
	w.Lock()
	defer w.Unlock()
	if fsWatcher == nil {
		return
	}
	fileWatchMap.Delete(w.path)
	err := fsWatcher.Remove(w.path)
	if err != nil {
		w.l.Error(err)
	}
}

func (w *fileWatcher) Dispose() {
	w.Stop()
}

func (w *dockerWatcher) Start() {
	w.Lock()
	defer w.Unlock()
	dockerWatchMap.Set(w.name, w)
	w.wg.Add(1)
	go w.watch()
}

func (w *dockerWatcher) Stop() {
	w.Lock()
	defer w.Unlock()
	if w.stopCh == nil {
		return
	}
	close(w.stopCh)
	w.wg.Wait()
	w.stopCh = nil
	dockerWatchMap.Delete(w.name)
}

func (w *dockerWatcher) Dispose() {
	w.Stop()
	w.client.Close()
}

func InitFSWatcher() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		wlog.Errorf("unable to create file watcher: %v", err)
		return
	}
	fsWatcher = w
	fsWatcherWg.Add(1)
	go watchFiles()
}

func StopFSWatcher() {
	close(fsWatcherStop)
	fsWatcherWg.Wait()
}

func StopDockerWatcher() {
	ParallelForEachValue(
		dockerWatchMap.Iterator(),
		(*dockerWatcher).Dispose,
	)
}

func watchFiles() {
	defer fsWatcher.Close()
	defer fsWatcherWg.Done()

	for {
		select {
		case <-fsWatcherStop:
			return
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
				w.l.Info("file changed")
				go w.onChange()
			case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
				w.l.Info("file renamed / deleted")
				go w.onDelete()
			}
		case err := <-fsWatcher.Errors:
			wlog.Error(err)
		}
	}
}

func (w *dockerWatcher) watch() {
	defer w.wg.Done()

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
		case <-w.stopCh:
			return
		case msg := <-msgChan:
			w.l.Infof("container %s %s", msg.Actor.Attributes["name"], msg.Action)
			go w.onChange()
		case err := <-errChan:
			switch {
			case client.IsErrConnectionFailed(err):
				w.l.Error(NewNestedError("connection failed").Subject(w.name))
			case client.IsErrNotFound(err):
				w.l.Error(NewNestedError("endpoint not found").Subject(w.name))
			default:
				w.l.Error(NewNestedErrorFrom(err).Subject(w.name))
			}
			time.Sleep(1 * time.Second)
			msgChan, errChan = listen()
		}
	}
}

type (
	FileWatcherMap   = SafeMap[string, *fileWatcher]
	DockerWatcherMap = SafeMap[string, *dockerWatcher]
)

var fsWatcher *fsnotify.Watcher
var (
	fileWatchMap   FileWatcherMap   = NewSafeMapOf[FileWatcherMap]()
	dockerWatchMap DockerWatcherMap = NewSafeMapOf[DockerWatcherMap]()
)
var (
	fsWatcherStop = make(chan struct{}, 1)
)
var (
	fsWatcherWg sync.WaitGroup
)
