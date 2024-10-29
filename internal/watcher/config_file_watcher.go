package watcher

import (
	"sync"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/task"
)

var (
	configDirWatcher   *DirWatcher
	configDirWatcherMu sync.Mutex
)

// create a new file watcher for file under ConfigBasePath.
func NewConfigFileWatcher(filename string) Watcher {
	configDirWatcherMu.Lock()
	defer configDirWatcherMu.Unlock()
	if configDirWatcher == nil {
		configDirWatcher = NewDirectoryWatcher(task.GlobalTask("config watcher"), common.ConfigBasePath)
	}
	return configDirWatcher.Add(filename)
}
