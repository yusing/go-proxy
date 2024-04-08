package main

import (
	"strings"
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

func (p *Provider) newWatcher() *watcherBase {
	return &watcherBase{
		onChange: p.ReloadRoutes,
		l:        p.l,
	}
}

func (p *Provider) NewFileWatcher() Watcher {
	return &fileWatcher{
		watcherBase: p.newWatcher(),
		path:        p.GetFilePath(),
		onDelete:    p.StopAllRoutes,
	}
}

func (p *Provider) NewDockerWatcher(c *client.Client) Watcher {
	return &dockerWatcher{
		watcherBase: p.newWatcher(),
		client:      c,
		stopCh:      make(chan struct{}, 1),
	}
}

func (c *config) newWatcher() *watcherBase {
	return &watcherBase{
		onChange: c.MustReload,
		l:        c.l,
	}
}

func (c *config) NewFileWatcher() Watcher {
	return &fileWatcher{
		watcherBase: c.newWatcher(),
		path:        c.reader.(*FileReader).Path,
		onDelete:    func() { c.l.Fatal("config file deleted") },
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
	dockerWatchMap.Set(w.client.DaemonHost(), w)
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
	dockerWatchMap.Delete(w.client.DaemonHost())
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
				w.l.Info("file changed: ", event.Name)
				go w.onChange()
			case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
				w.l.Info("file renamed / deleted: ", event.Name)
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
			containerName := msg.Actor.Attributes["name"]
			if strings.HasPrefix(containerName, "buildx_buildkit_builder-") {
				continue
			}
			w.l.Infof("container %s %s", containerName, msg.Action)
			go w.onChange()
		case err := <-errChan:
			switch {
			case client.IsErrConnectionFailed(err):
				w.l.Error("watcher: connection failed")
			case client.IsErrNotFound(err):
				w.l.Error("watcher: endpoint not found")
			default:
				w.l.Errorf("watcher: %v", err)
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
