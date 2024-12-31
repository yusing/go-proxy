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

		buf            bytes.Buffer
		bufPool        sync.Pool
		flushThreshold int
		flushMu        sync.Mutex

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
	}
)

var logger = logging.With().Str("module", "accesslog").Logger()

func NewAccessLogger(parent task.Parent, io AccessLogIO, cfg *Config) *AccessLogger {
	l := &AccessLogger{
		task: parent.Subtask("accesslog"),
		cfg:  cfg,
		io:   io,
	}
	if cfg.BufferSize < 1024 {
		cfg.BufferSize = DefaultBufferSize
	}

	fmt := &CommonFormatter{cfg: &l.cfg.Fields, GetTimeNow: time.Now}
	switch l.cfg.Format {
	case FormatCommon:
		l.Formatter = fmt
	case FormatCombined:
		l.Formatter = (*CombinedFormatter)(fmt)
	case FormatJSON:
		l.Formatter = (*JSONFormatter)(fmt)
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

	l.flushMu.Lock()
	l.buf.Write(line.Bytes())
	line.Reset()
	l.bufPool.Put(line)
	l.flushMu.Unlock()
}

func (l *AccessLogger) LogError(req *http.Request, err error) {
	l.Log(req, &http.Response{StatusCode: http.StatusInternalServerError, Status: err.Error()})
}

func (l *AccessLogger) Config() *Config {
	return l.cfg
}

// func (l *AccessLogger) Rotate() error {
// 	if l.cfg.Retention == nil {
// 		return nil
// 	}
// 	l.io.Lock()
// 	defer l.io.Unlock()

// 	return l.cfg.Retention.rotateLogFile(l.io)
// }

func (l *AccessLogger) Flush(force bool) {
	l.flushMu.Lock()
	if force || l.buf.Len() >= l.flushThreshold {
		l.writeLine(l.buf.Bytes())
		l.buf.Reset()
	}
	l.flushMu.Unlock()
}

func (l *AccessLogger) handleErr(err error) {
	E.LogError("failed to write access log", err, &logger)
}

func (l *AccessLogger) start() {
	defer func() {
		if l.buf.Len() > 0 { // flush last
			l.writeLine(l.buf.Bytes())
		}
		l.io.Close()
		l.task.Finish(nil)
	}()

	// periodic + threshold flush
	flushTicker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-l.task.Context().Done():
			return
		case <-flushTicker.C:
			l.Flush(true)
		default:
			l.Flush(false)
		}
	}
}

func (l *AccessLogger) writeLine(line []byte) {
	l.io.Lock() // prevent write on log rotation
	_, err := l.io.Write(line)
	l.io.Unlock()
	if err != nil {
		l.handleErr(err)
	}
}
