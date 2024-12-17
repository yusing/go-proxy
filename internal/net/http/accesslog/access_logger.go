package accesslog

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
)

type (
	AccessLogger struct {
		parent *task.Task
		buf    chan []byte
		cfg    *Config
		w      io.WriteCloser
		Formatter
	}

	Formatter interface {
		// Format writes a log line to line without a trailing newline
		Format(line *bytes.Buffer, req *http.Request, res *http.Response)
	}
)

var logger = logging.With().Str("module", "accesslog").Logger()

var TestTimeNow = time.Now().Format(logTimeFormat)

const logTimeFormat = "02/Jan/2006:15:04:05 -0700"

func NewFileAccessLogger(parent *task.Task, cfg *Config) (*AccessLogger, error) {
	f, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return NewAccessLogger(parent, f, cfg), nil
}

func NewAccessLogger(parent *task.Task, w io.WriteCloser, cfg *Config) *AccessLogger {
	l := &AccessLogger{
		parent: parent,
		cfg:    cfg,
		w:      w,
	}
	fmt := CommonFormatter{cfg: &l.cfg.Fields}
	switch l.cfg.Format {
	case FormatCommon:
		l.Formatter = fmt
	case FormatCombined:
		l.Formatter = CombinedFormatter{CommonFormatter: fmt}
	case FormatJSON:
		l.Formatter = JSONFormatter{CommonFormatter: fmt}
	}
	if cfg.BufferSize == 0 {
		cfg.BufferSize = DefaultBufferSize
	}
	l.buf = make(chan []byte, cfg.BufferSize)
	go l.start()
	return l
}

func timeNow() string {
	if !common.IsTest {
		return time.Now().Format(logTimeFormat)
	}
	return TestTimeNow
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

	var line bytes.Buffer
	l.Format(&line, req, res)
	line.WriteRune('\n')

	select {
	case <-l.parent.Context().Done():
		return
	default:
		l.buf <- line.Bytes()
	}
}

func (l *AccessLogger) LogError(req *http.Request, err error) {
	l.Log(req, &http.Response{StatusCode: http.StatusInternalServerError, Status: err.Error()})
}

func (l *AccessLogger) close() {
	close(l.buf)
	l.w.Close()
}

func (l *AccessLogger) handleErr(err error) {
	E.LogError("failed to write access log", err, &logger)
}

func (l *AccessLogger) start() {
	task := l.parent.Subtask("access log flusher")
	defer task.Finish("done")
	defer l.close()

	for {
		select {
		case <-task.Context().Done():
			return
		default:
			for line := range l.buf {
				_, err := l.w.Write(line)
				if err != nil {
					l.handleErr(err)
				}
			}
		}
	}
}
