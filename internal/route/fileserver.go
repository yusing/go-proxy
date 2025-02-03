package route

import (
	"net/http"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/http/accesslog"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	metricslogger "github.com/yusing/go-proxy/internal/net/http/middleware/metrics_logger"
	"github.com/yusing/go-proxy/internal/route/routes"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"

	E "github.com/yusing/go-proxy/internal/error"
)

type (
	FileServer struct {
		*Route

		task         *task.Task
		middleware   *middleware.Middleware
		handler      http.Handler
		accessLogger *accesslog.AccessLogger
		startTime    time.Time
	}
)

func handler(root string) http.Handler {
	return http.FileServer(http.Dir(root))
}

func NewFileServer(base *Route) (*FileServer, E.Error) {
	s := &FileServer{Route: base, handler: handler(base.Root)}

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

	pathPatterns := s.PathPatterns
	switch {
	case len(pathPatterns) == 0:
	case len(pathPatterns) == 1 && pathPatterns[0] == "/":
	default:
		mux := gphttp.NewServeMux()
		patErrs := E.NewBuilder("invalid path pattern(s)")
		for _, p := range pathPatterns {
			patErrs.Add(mux.Handle(p, s.handler))
		}
		if err := patErrs.Error(); err != nil {
			s.task.Finish(err)
			return err
		}
		s.handler = mux
	}

	if s.middleware != nil {
		s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.middleware.ServeHTTP(s.handler.ServeHTTP, w, r)
		})
	}

	if s.UseAccessLog() {
		var err error
		s.accessLogger, err = accesslog.NewFileAccessLogger(s.task, s.AccessLog)
		if err != nil {
			s.task.Finish(err)
			return E.Wrap(err)
		}
	}

	if common.PrometheusEnabled {
		metricsLogger := metricslogger.NewMetricsLogger(s.TargetName())
		s.handler = metricsLogger.GetHandler(s.handler)
		s.task.OnCancel("reset_metrics", metricsLogger.ResetMetrics)
	}

	routes.SetHTTPRoute(s.TargetName(), s)
	s.task.OnCancel("entrypoint_remove_route", func() {
		routes.DeleteHTTPRoute(s.TargetName())
	})
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
	s.handler.ServeHTTP(w, req)
	if s.accessLogger != nil {
		s.accessLogger.Log(req, req.Response)
	}
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
