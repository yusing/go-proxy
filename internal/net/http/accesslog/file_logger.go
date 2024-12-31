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

func NewFileAccessLogger(parent task.Parent, cfg *Config) (*AccessLogger, error) {
	f, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("access log open error: %w", err)
	}
	return NewAccessLogger(parent, &File{File: f}, cfg), nil
}
