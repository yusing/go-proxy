package route

import (
	"net/http"
	"time"

	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"

	E "github.com/yusing/go-proxy/internal/error"
)

type (
	FileServer struct {
		*Route

		task       *task.Task
		middleware *middleware.Middleware
		handler    http.Handler
		startTime  time.Time
	}
)

func handler(root string) http.Handler {
	return http.FileServer(http.Dir(root))
}

func NewFileServer(base *Route) (*FileServer, E.Error) {
	s := &FileServer{Route: base}
	s.handler = handler(s.Root)

	if len(s.Rules) > 0 {
		s.handler = s.Rules.BuildHandler(s.Alias, s.handler)
	}

	if len(s.Middlewares) > 0 {
		mid, err := middleware.BuildMiddlewareFromMap(s.Alias, s.Middlewares)
		if err != nil {
			return nil, err
		}
		s.middleware = mid
	}

	return s, nil
}

// Start implements task.TaskStarter.
func (s *FileServer) Start(parent task.Parent) E.Error {
	s.startTime = time.Now()
	s.task = parent.Subtask("fileserver."+s.Name(), false)
	return nil
}

func (s *FileServer) Task() *task.Task {
	return s.task
}

// Finish implements task.TaskFinisher.
func (s *FileServer) Finish(reason any) {
	s.task.Finish(reason)
}

// ServeHTTP implements http.Handler.
func (s *FileServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.middleware != nil {
		s.middleware.ServeHTTP(s.handler.ServeHTTP, w, req)
	}
	s.handler.ServeHTTP(w, req)
}

// Status implements health.HealthMonitor.
func (s *FileServer) Status() health.Status {
	return health.StatusHealthy
}

// Uptime implements health.HealthMonitor.
func (s *FileServer) Uptime() time.Duration {
	return time.Since(s.startTime)
}

// Latency implements health.HealthMonitor.
func (s *FileServer) Latency() time.Duration {
	return 0
}

// MarshalJSON implements json.Marshaler.
func (s *FileServer) MarshalJSON() ([]byte, error) {
	return (&monitor.JSONRepresentation{
		Name:     s.Alias,
		Config:   nil,
		Status:   s.Status(),
		Started:  s.startTime,
		Uptime:   s.Uptime(),
		Latency:  s.Latency(),
		LastSeen: time.Now(),
		Detail:   "",
		URL:      nil,
	}).MarshalJSON()
}

func (s *FileServer) String() string {
	return "FileServer " + s.Alias
}

func (s *FileServer) Name() string {
	return s.Alias
}
