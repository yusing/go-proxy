package accesslog

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
)

type (
	AccessLogger struct {
		task *task.Task
		cfg  *Config
		io   AccessLogIO

		buf     bytes.Buffer // buffer for non-flushed log
		bufMu   sync.Mutex   // protect buf
		bufPool sync.Pool    // buffer pool for formatting a single log line

		flushThreshold int

		Formatter
	}

	AccessLogIO interface {
		io.ReadWriteCloser
		io.ReadWriteSeeker
		io.ReaderAt
		sync.Locker
		Name() string // file name or path
		Truncate(size int64) error
	}

	Formatter interface {
		// Format writes a log line to line without a trailing newline
		Format(line *bytes.Buffer, req *http.Request, res *http.Response)
		SetGetTimeNow(getTimeNow func() time.Time)
	}
)

func NewAccessLogger(parent task.Parent, io AccessLogIO, cfg *Config) *AccessLogger {
	l := &AccessLogger{
		task: parent.Subtask("accesslog"),
		cfg:  cfg,
		io:   io,
	}
	if cfg.BufferSize < 1024 {
		cfg.BufferSize = DefaultBufferSize
	}

	fmt := CommonFormatter{cfg: &l.cfg.Fields, GetTimeNow: time.Now}
	switch l.cfg.Format {
	case FormatCommon:
		l.Formatter = &fmt
	case FormatCombined:
		l.Formatter = &CombinedFormatter{fmt}
	case FormatJSON:
		l.Formatter = &JSONFormatter{fmt}
	default: // should not happen, validation has done by validate tags
		panic("invalid access log format")
	}

	l.flushThreshold = int(cfg.BufferSize * 4 / 5) // 80%
	l.buf.Grow(int(cfg.BufferSize))
	l.bufPool.New = func() any {
		return new(bytes.Buffer)
	}
	go l.start()
	return l
}

func (l *AccessLogger) checkKeep(req *http.Request, res *http.Response) bool {
	if !l.cfg.Filters.StatusCodes.CheckKeep(req, res) ||
		!l.cfg.Filters.Method.CheckKeep(req, res) ||
		!l.cfg.Filters.Headers.CheckKeep(req, res) ||
		!l.cfg.Filters.CIDR.CheckKeep(req, res) {
		return false
	}
	return true
}

func (l *AccessLogger) Log(req *http.Request, res *http.Response) {
	if !l.checkKeep(req, res) {
		return
	}

	line := l.bufPool.Get().(*bytes.Buffer)
	l.Format(line, req, res)
	line.WriteRune('\n')

	l.bufMu.Lock()
	l.buf.Write(line.Bytes())
	line.Reset()
	l.bufPool.Put(line)
	l.bufMu.Unlock()
}

func (l *AccessLogger) LogError(req *http.Request, err error) {
	l.Log(req, &http.Response{StatusCode: http.StatusInternalServerError, Status: err.Error()})
}

func (l *AccessLogger) Config() *Config {
	return l.cfg
}

func (l *AccessLogger) Rotate() error {
	if l.cfg.Retention == nil {
		return nil
	}
	l.io.Lock()
	defer l.io.Unlock()

	return l.cfg.Retention.rotateLogFile(l.io)
}

func (l *AccessLogger) Flush(force bool) {
	if l.buf.Len() == 0 {
		return
	}
	if force || l.buf.Len() >= l.flushThreshold {
		l.bufMu.Lock()
		l.write(l.buf.Bytes())
		l.buf.Reset()
		l.bufMu.Unlock()
	}
}

func (l *AccessLogger) handleErr(err error) {
	E.LogError("failed to write access log", err)
}

func (l *AccessLogger) start() {
	defer func() {
		if l.buf.Len() > 0 { // flush last
			l.write(l.buf.Bytes())
		}
		l.io.Close()
		l.task.Finish(nil)
	}()

	// periodic flush + threshold flush
	periodic := time.NewTicker(5 * time.Second)
	threshold := time.NewTicker(time.Second)
	defer periodic.Stop()
	defer threshold.Stop()

	for {
		select {
		case <-l.task.Context().Done():
			return
		case <-periodic.C:
			l.Flush(true)
		case <-threshold.C:
			l.Flush(false)
		}
	}
}

func (l *AccessLogger) write(data []byte) {
	l.io.Lock() // prevent concurrent write, i.e. log rotation, other access loggers
	_, err := l.io.Write(data)
	l.io.Unlock()
	if err != nil {
		l.handleErr(err)
	} else {
		logging.Debug().Msg("access log flushed to " + l.io.Name())
	}
}
