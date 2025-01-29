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
	"github.com/yusing/go-proxy/internal/route/entry"
	"github.com/yusing/go-proxy/internal/route/routes"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"
)

type (
	HTTPRoute struct {
		*entry.ReverseProxyEntry

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

func NewHTTPRoute(entry *entry.ReverseProxyEntry) (impl, E.Error) {
	trans := gphttp.DefaultTransport
	httpConfig := entry.Raw.HTTPConfig

	if httpConfig.NoTLSVerify {
		trans = gphttp.DefaultTransportNoTLS
	}
	if httpConfig.ResponseHeaderTimeout > 0 {
		trans = trans.Clone()
		trans.ResponseHeaderTimeout = httpConfig.ResponseHeaderTimeout
	}

	service := entry.TargetName()
	rp := reverseproxy.NewReverseProxy(service, entry.URL, trans)

	if len(entry.Raw.Middlewares) > 0 {
		err := middleware.PatchReverseProxy(rp, entry.Raw.Middlewares)
		if err != nil {
			return nil, err
		}
	}

	r := &HTTPRoute{
		ReverseProxyEntry: entry,
		rp:                rp,
		l: logging.With().
			Str("type", entry.URL.Scheme).
			Str("name", service).
			Logger(),
	}
	return r, nil
}

func (r *HTTPRoute) String() string {
	return r.TargetName()
}

// Start implements task.TaskStarter.
func (r *HTTPRoute) Start(parent task.Parent) E.Error {
	if entry.ShouldNotServe(r) {
		return nil
	}

	r.task = parent.Subtask("http."+r.TargetName(), false)

	switch {
	case entry.UseIdleWatcher(r):
		waker, err := idlewatcher.NewHTTPWaker(parent, r.ReverseProxyEntry, r.rp)
		if err != nil {
			r.task.Finish(err)
			return err
		}
		r.handler = waker
		r.HealthMon = waker
	case entry.UseHealthCheck(r):
		if entry.IsDocker(r) {
			client, err := docker.ConnectClient(r.Idlewatcher.DockerHost)
			if err == nil {
				fallback := monitor.NewHTTPHealthChecker(r.rp.TargetURL, r.Raw.HealthCheck)
				r.HealthMon = monitor.NewDockerHealthMonitor(client, r.Idlewatcher.ContainerID, r.TargetName(), r.Raw.HealthCheck, fallback)
				r.task.OnCancel("close_docker_client", client.Close)
			}
		}
		if r.HealthMon == nil {
			r.HealthMon = monitor.NewHTTPHealthMonitor(r.rp.TargetURL, r.Raw.HealthCheck)
		}
	}

	if entry.UseAccessLog(r) {
		var err error
		r.rp.AccessLogger, err = accesslog.NewFileAccessLogger(r.task, r.Raw.AccessLog)
		if err != nil {
			r.task.Finish(err)
			return E.From(err)
		}
	}

	if r.handler == nil {
		pathPatterns := r.Raw.PathPatterns
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

	if len(r.Raw.Rules) > 0 {
		r.handler = r.Raw.Rules.BuildHandler(r.TargetName(), r.handler)
	}

	if r.HealthMon != nil {
		if err := r.HealthMon.Start(r.task); err != nil {
			E.LogWarn("health monitor error", err, &r.l)
		}
	}

	if entry.UseLoadBalance(r) {
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
func (r *HTTPRoute) Task() *task.Task {
	return r.task
}

// Finish implements task.TaskFinisher.
func (r *HTTPRoute) Finish(reason any) {
	r.task.Finish(reason)
}

func (r *HTTPRoute) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *HTTPRoute) HealthMonitor() health.HealthMonitor {
	return r.HealthMon
}

func (r *HTTPRoute) addToLoadBalancer(parent task.Parent) {
	var lb *loadbalancer.LoadBalancer
	cfg := r.Raw.LoadBalance
	l, ok := routes.GetHTTPRoute(cfg.Link)
	var linked *HTTPRoute
	if ok {
		linked = l.(*HTTPRoute)
		lb = linked.loadBalancer
		lb.UpdateConfigIfNeeded(cfg)
		if linked.Raw.Homepage.IsEmpty() && !r.Raw.Homepage.IsEmpty() {
			linked.Raw.Homepage = r.Raw.Homepage
		}
	} else {
		lb = loadbalancer.New(cfg)
		if err := lb.Start(parent); err != nil {
			panic(err) // should always return nil
		}
		linked = &HTTPRoute{
			ReverseProxyEntry: &entry.ReverseProxyEntry{
				Raw: &route.RawEntry{
					Alias:    cfg.Link,
					Homepage: r.Raw.Homepage,
				},
			},
			HealthMon:    lb,
			loadBalancer: lb,
			handler:      lb,
		}
		routes.SetHTTPRoute(cfg.Link, linked)
	}
	r.loadBalancer = lb
	r.server = loadbalance.NewServer(r.task.Name(), r.rp.TargetURL, r.Raw.LoadBalance.Weight, r.handler, r.HealthMon)
	lb.AddServer(r.server)
	r.task.OnCancel("lb_remove_server", func() {
		lb.RemoveServer(r.server)
	})
}
