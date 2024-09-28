package watcher

import (
	"context"
	"sync"

	"github.com/yusing/go-proxy/internal/common"
)

var configDirWatcher *dirWatcher
var configDirWatcherMu sync.Mutex

// create a new file watcher for file under ConfigBasePath
func NewConfigFileWatcher(filename string) Watcher {
	configDirWatcherMu.Lock()
	defer configDirWatcherMu.Unlock()
	if configDirWatcher == nil {
		configDirWatcher = NewDirectoryWatcher(context.Background(), common.ConfigBasePath)
	}
	return configDirWatcher.Add(filename)
}
