package watcher

import (
	"context"
	"path"

	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
)

type fileWatcher struct {
	filename string
}

func NewFileWatcher(filename string) Watcher {
	if path.Base(filename) != filename {
		panic("filename must be a relative path")
	}
	return &fileWatcher{filename: filename}
}

func (f *fileWatcher) Events(ctx context.Context) (<-chan Event, <-chan E.NestedError) {
	return fwHelper.Add(ctx, f)
}

func InitFileWatcherHelper() {
	fwHelper = newFileWatcherHelper(common.ConfigBasePath)
}

var fwHelper *fileWatcherHelper
