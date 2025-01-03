package accesslog

import (
	"fmt"
	"os"
	"sync"

	"github.com/yusing/go-proxy/internal/task"
)

type File struct {
	*os.File
	sync.Mutex
}

var (
	openedFiles   = make(map[string]AccessLogIO)
	openedFilesMu sync.Mutex
)

func NewFileAccessLogger(parent task.Parent, cfg *Config) (*AccessLogger, error) {
	openedFilesMu.Lock()

	var io AccessLogIO
	if opened, ok := openedFiles[cfg.Path]; ok {
		io = opened
	} else {
		f, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("access log open error: %w", err)
		}
		io = &File{File: f}
		openedFiles[cfg.Path] = io
	}

	openedFilesMu.Unlock()
	return NewAccessLogger(parent, io, cfg), nil
}
