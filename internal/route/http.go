package route

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher"
	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	loadbalance "github.com/yusing/go-proxy/internal/net/http/loadbalancer/types"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
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
		server       *loadbalancer.Server
		handler      http.Handler
		rp           *gphttp.ReverseProxy

		task *task.Task

		l zerolog.Logger
	}
)

// var globalMux    = http.NewServeMux() // TODO: support regex subdomain matching.

func NewHTTPRoute(entry *entry.ReverseProxyEntry) (impl, E.Error) {
	var trans *http.Transport
	if entry.Raw.NoTLSVerify {
		trans = gphttp.DefaultTransportNoTLS
	} else {
		trans = gphttp.DefaultTransport
	}

	service := entry.TargetName()
	rp := gphttp.NewReverseProxy(service, entry.URL, trans)

	if len(entry.Raw.Middlewares) > 0 {
		err := middleware.PatchReverseProxy(rp, entry.Raw.Middlewares)
		if err != nil {
			return nil, err
		}
	}

	r := &HTTPRoute{
		ReverseProxyEntry: entry,
		rp:                rp,
		l: logger.With().
			Str("type", entry.URL.Scheme).
			Str("name", service).
			Logger(),
	}
	return r, nil
}

func (r *HTTPRoute) String() string {
	return r.TargetName()
}

// Start implements*task.TaskStarter.
func (r *HTTPRoute) Start(providerSubtask *task.Task) E.Error {
	if entry.ShouldNotServe(r) {
		providerSubtask.Finish("should not serve")
		return nil
	}

	r.task = providerSubtask

	switch {
	case entry.UseIdleWatcher(r):
		wakerTask := providerSubtask.Parent().Subtask("waker for " + r.TargetName())
		waker, err := idlewatcher.NewHTTPWaker(wakerTask, r.ReverseProxyEntry, r.rp)
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
				r.HealthMon = monitor.NewDockerHealthMonitor(client, r.Idlewatcher.ContainerID, r.Raw.HealthCheck, fallback)
				r.task.OnCancel("close docker client", client.Close)
			}
		}
		if r.HealthMon == nil {
			r.HealthMon = monitor.NewHTTPHealthMonitor(r.rp.TargetURL, r.Raw.HealthCheck)
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

	if r.HealthMon != nil {
		healthMonTask := r.task.Subtask("health monitor")
		if err := r.HealthMon.Start(healthMonTask); err != nil {
			E.LogWarn("health monitor error", err, &r.l)
			healthMonTask.Finish(err)
		}
	}

	if entry.UseLoadBalance(r) {
		r.addToLoadBalancer()
	} else {
		routes.SetHTTPRoute(r.TargetName(), r)
		r.task.OnFinished("remove from route table", func() {
			routes.DeleteHTTPRoute(r.TargetName())
		})
	}

	if common.PrometheusEnabled {
		r.task.OnFinished("unreg metrics", r.rp.UnregisterMetrics)
	}
	return nil
}

// Finish implements*task.TaskFinisher.
func (r *HTTPRoute) Finish(reason any) {
	r.task.Finish(reason)
}

func (r *HTTPRoute) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *HTTPRoute) addToLoadBalancer() {
	var lb *loadbalancer.LoadBalancer
	cfg := r.Raw.LoadBalance
	l, ok := routes.GetHTTPRoute(cfg.Link)
	var linked *HTTPRoute
	if ok {
		linked = l.(*HTTPRoute)
		lb = linked.loadBalancer
		lb.UpdateConfigIfNeeded(cfg)
		if linked.Raw.Homepage == nil && r.Raw.Homepage != nil {
			linked.Raw.Homepage = r.Raw.Homepage
		}
	} else {
		lb = loadbalancer.New(cfg)
		lbTask := r.task.Parent().Subtask("loadbalancer " + cfg.Link)
		lbTask.OnCancel("remove lb from routes", func() {
			routes.DeleteHTTPRoute(cfg.Link)
		})
		if err := lb.Start(lbTask); err != nil {
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
	r.server = loadbalance.NewServer(r.task.String(), r.rp.TargetURL, r.Raw.LoadBalance.Weight, r.handler, r.HealthMon)
	lb.AddServer(r.server)
	r.task.OnCancel("remove server from lb", func() {
		lb.RemoveServer(r.server)
	})
}
