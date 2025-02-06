package route

import (
	"net/http"
	"path"
	"path/filepath"

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

		Health *monitor.FileServerHealthMonitor `json:"health"`

		task         *task.Task
		middleware   *middleware.Middleware
		handler      http.Handler
		accessLogger *accesslog.AccessLogger
	}
)

func handler(root string) http.Handler {
	return http.FileServer(http.Dir(root))
}

func NewFileServer(base *Route) (*FileServer, E.Error) {
	s := &FileServer{Route: base}

	s.Root = filepath.Clean(s.Root)
	if !path.IsAbs(s.Root) {
		return nil, E.New("`root` must be an absolute path")
	}

	s.handler = handler(s.Root)

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
	s.task = parent.Subtask("fileserver."+s.TargetName(), false)

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

	if s.UseHealthCheck() {
		s.Health = monitor.NewFileServerHealthMonitor(s.TargetName(), s.HealthCheck, s.Root)
		if err := s.Health.Start(s.task); err != nil {
			return err
		}
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

func (s *FileServer) HealthMonitor() health.HealthMonitor {
	return s.Health
}
