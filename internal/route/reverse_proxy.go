package route

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/api/v1/favicon"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/http/accesslog"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	loadbalance "github.com/yusing/go-proxy/internal/net/http/loadbalancer/types"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/net/http/reverseproxy"
	"github.com/yusing/go-proxy/internal/route/routes"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"
)

type (
	ReveseProxyRoute struct {
		*Route

		HealthMon health.HealthMonitor `json:"health,omitempty"`

		loadBalancer *loadbalancer.LoadBalancer
		server       loadbalancer.Server
		handler      http.Handler
		rp           *reverseproxy.ReverseProxy

		task *task.Task

		l zerolog.Logger
	}
)

// var globalMux    = http.NewServeMux() // TODO: support regex subdomain matching.

func NewReverseProxyRoute(base *Route) (*ReveseProxyRoute, E.Error) {
	trans := gphttp.DefaultTransport
	httpConfig := base.HTTPConfig

	if httpConfig.NoTLSVerify {
		trans = gphttp.DefaultTransportNoTLS
	}
	if httpConfig.ResponseHeaderTimeout > 0 {
		trans = trans.Clone()
		trans.ResponseHeaderTimeout = httpConfig.ResponseHeaderTimeout
	}

	service := base.TargetName()
	rp := reverseproxy.NewReverseProxy(service, base.pURL, trans)

	if len(base.Middlewares) > 0 {
		err := middleware.PatchReverseProxy(rp, base.Middlewares)
		if err != nil {
			return nil, err
		}
	}

	r := &ReveseProxyRoute{
		Route: base,
		rp:    rp,
		l: logging.With().
			Str("type", string(base.Scheme)).
			Str("name", service).
			Logger(),
	}
	return r, nil
}

func (r *ReveseProxyRoute) String() string {
	return r.TargetName()
}

// Start implements task.TaskStarter.
func (r *ReveseProxyRoute) Start(parent task.Parent) E.Error {
	r.task = parent.Subtask("http."+r.TargetName(), false)

	switch {
	case r.UseIdleWatcher():
		waker, err := idlewatcher.NewHTTPWaker(parent, r, r.rp)
		if err != nil {
			r.task.Finish(err)
			return err
		}
		r.handler = waker
		r.HealthMon = waker
	case r.UseHealthCheck():
		if r.IsDocker() {
			client, err := docker.ConnectClient(r.idlewatcher.DockerHost)
			if err == nil {
				fallback := monitor.NewHTTPHealthChecker(r.rp.TargetURL, r.HealthCheck)
				r.HealthMon = monitor.NewDockerHealthMonitor(client, r.idlewatcher.ContainerID, r.TargetName(), r.HealthCheck, fallback)
				r.task.OnCancel("close_docker_client", client.Close)
			}
		}
		if r.HealthMon == nil {
			r.HealthMon = monitor.NewHTTPHealthMonitor(r.rp.TargetURL, r.HealthCheck)
		}
	}

	if r.UseAccessLog() {
		var err error
		r.rp.AccessLogger, err = accesslog.NewFileAccessLogger(r.task, r.AccessLog)
		if err != nil {
			r.task.Finish(err)
			return E.From(err)
		}
	}

	if r.handler == nil {
		pathPatterns := r.PathPatterns
		switch {
		case len(pathPatterns) == 0:
			r.handler = r.rp
		case len(pathPatterns) == 1 && pathPatterns[0] == "/":
			r.handler = r.rp
		default:
			logging.Warn().
				Str("route", r.TargetName()).
				Msg("`path_patterns` is deprecated. Use `rules` instead.")
			mux := gphttp.NewServeMux()
			patErrs := E.NewBuilder("invalid path pattern(s)")
			for _, p := range pathPatterns {
				patErrs.Add(mux.HandleFunc(p, r.rp.HandlerFunc))
			}
			if err := patErrs.Error(); err != nil {
				r.task.Finish(err)
				return err
			}
			r.handler = mux
		}
	}

	if len(r.Rules) > 0 {
		r.handler = r.Rules.BuildHandler(r.TargetName(), r.handler)
	}

	if r.HealthMon != nil {
		if err := r.HealthMon.Start(r.task); err != nil {
			E.LogWarn("health monitor error", err, &r.l)
		}
	}

	if r.UseLoadBalance() {
		r.addToLoadBalancer(parent)
	} else {
		routes.SetHTTPRoute(r.TargetName(), r)
		r.task.OnCancel("entrypoint_remove_route", func() {
			routes.DeleteHTTPRoute(r.TargetName())
		})
	}

	if common.PrometheusEnabled {
		r.task.OnCancel("metrics_cleanup", r.rp.UnregisterMetrics)
	}

	r.task.OnCancel("reset_favicon", func() { favicon.PruneRouteIconCache(r) })
	return nil
}

// Task implements task.TaskStarter.
func (r *ReveseProxyRoute) Task() *task.Task {
	return r.task
}

// Finish implements task.TaskFinisher.
func (r *ReveseProxyRoute) Finish(reason any) {
	r.task.Finish(reason)
}

func (r *ReveseProxyRoute) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *ReveseProxyRoute) HealthMonitor() health.HealthMonitor {
	return r.HealthMon
}

func (r *ReveseProxyRoute) addToLoadBalancer(parent task.Parent) {
	var lb *loadbalancer.LoadBalancer
	cfg := r.LoadBalance
	l, ok := routes.GetHTTPRoute(cfg.Link)
	var linked *ReveseProxyRoute
	if ok {
		linked = l.(*ReveseProxyRoute)
		lb = linked.loadBalancer
		lb.UpdateConfigIfNeeded(cfg)
		if linked.Homepage.IsEmpty() && !r.Homepage.IsEmpty() {
			linked.Homepage = r.Homepage
		}
	} else {
		lb = loadbalancer.New(cfg)
		if err := lb.Start(parent); err != nil {
			panic(err) // should always return nil
		}
		linked = &ReveseProxyRoute{
			Route: &Route{
				Alias:    cfg.Link,
				Homepage: r.Homepage,
			},
			HealthMon:    lb,
			loadBalancer: lb,
			handler:      lb,
		}
		routes.SetHTTPRoute(cfg.Link, linked)
	}
	r.loadBalancer = lb
	r.server = loadbalance.NewServer(r.task.Name(), r.rp.TargetURL, r.LoadBalance.Weight, r.handler, r.HealthMon)
	lb.AddServer(r.server)
	r.task.OnCancel("lb_remove_server", func() {
		lb.RemoveServer(r.server)
	})
}
