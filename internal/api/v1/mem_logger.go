package v1

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type logEntryRange struct {
	Start, End int
}

type memLogger struct {
	bytes.Buffer
	sync.Mutex
	connChans F.Map[chan *logEntryRange, struct{}]
}

const (
	maxMemLogSize         = 16 * 1024
	truncateSize          = maxMemLogSize / 2
	initialWriteChunkSize = 4 * 1024
)

var memLoggerInstance = &memLogger{
	connChans: F.NewMapOf[chan *logEntryRange, struct{}](),
}

func init() {
	if !common.EnableLogStreaming {
		return
	}
	memLoggerInstance.Grow(maxMemLogSize)

	if common.DebugMemLogger {
		ticker := time.NewTicker(1 * time.Second)

		go func() {
			defer ticker.Stop()

			for {
				select {
				case <-task.RootContextCanceled():
					return
				case <-ticker.C:
					logging.Info().Msgf("mem logger size: %d, active conns: %d",
						memLoggerInstance.Len(),
						memLoggerInstance.connChans.Size())
				}
			}
		}()
	}
}

func LogsWS() func(config config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	return memLoggerInstance.ServeHTTP
}

func MemLogger() io.Writer {
	return memLoggerInstance
}

func (m *memLogger) Write(p []byte) (n int, err error) {
	m.Lock()

	if m.Len() > maxMemLogSize {
		m.Truncate(truncateSize)
	}

	pos := m.Buffer.Len()
	n = len(p)
	_, err = m.Buffer.Write(p)
	if err != nil {
		m.Unlock()
		return
	}

	if m.connChans.Size() > 0 {
		m.Unlock()
		timeout := time.NewTimer(1 * time.Second)
		defer timeout.Stop()

		m.connChans.Range(func(ch chan *logEntryRange, _ struct{}) bool {
			select {
			case ch <- &logEntryRange{pos, pos + n}:
				return true
			case <-timeout.C:
				logging.Warn().Msg("mem logger: timeout logging to channel")
				return false
			}
		})
		return
	}

	m.Unlock()
	return
}

func (m *memLogger) ServeHTTP(config config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	conn, err := utils.InitiateWS(config, w, r)
	if err != nil {
		utils.HandleErr(w, r, err)
		return
	}

	logCh := make(chan *logEntryRange)
	m.connChans.Store(logCh, struct{}{})

	/* trunk-ignore(golangci-lint/errcheck) */
	defer func() {
		_ = conn.CloseNow()
		m.connChans.Delete(logCh)
		close(logCh)
	}()

	if err := m.wsInitial(r.Context(), conn); err != nil {
		utils.HandleErr(w, r, err)
		return
	}

	m.wsStreamLog(r.Context(), conn, logCh)
}

func (m *memLogger) writeBytes(ctx context.Context, conn *websocket.Conn, b []byte) error {
	return conn.Write(ctx, websocket.MessageText, b)
}

func (m *memLogger) wsInitial(ctx context.Context, conn *websocket.Conn) error {
	m.Lock()
	defer m.Unlock()

	return m.writeBytes(ctx, conn, m.Buffer.Bytes())
}

func (m *memLogger) wsStreamLog(ctx context.Context, conn *websocket.Conn, ch <-chan *logEntryRange) {
	for {
		select {
		case <-ctx.Done():
			return
		case logRange := <-ch:
			m.Lock()
			msg := m.Buffer.Bytes()[logRange.Start:logRange.End]
			err := m.writeBytes(ctx, conn, msg)
			m.Unlock()
			if err != nil {
				return
			}
		}
	}
}
