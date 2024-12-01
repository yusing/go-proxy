package route

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
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

		task task.Task

		l zerolog.Logger
	}

	SubdomainKey = route.Alias
)

// var globalMux    = http.NewServeMux() // TODO: support regex subdomain matching.

func NewHTTPRoute(entry *entry.ReverseProxyEntry) (impl, E.Error) {
	var trans *http.Transport
	if entry.NoTLSVerify {
		trans = gphttp.DefaultTransportNoTLS
	} else {
		trans = gphttp.DefaultTransport
	}

	service := string(entry.Alias)
	rp := gphttp.NewReverseProxy(service, entry.URL, trans)

	if len(entry.Middlewares) > 0 {
		err := middleware.PatchReverseProxy(rp, entry.Middlewares)
		if err != nil {
			return nil, err
		}
	}

	r := &HTTPRoute{
		ReverseProxyEntry: entry,
		rp:                rp,
		l: logger.With().
			Str("type", string(entry.Scheme)).
			Str("name", string(entry.Alias)).
			Logger(),
	}
	return r, nil
}

func (r *HTTPRoute) String() string {
	return string(r.Alias)
}

// Start implements task.TaskStarter.
func (r *HTTPRoute) Start(providerSubtask task.Task) E.Error {
	if entry.ShouldNotServe(r) {
		providerSubtask.Finish("should not serve")
		return nil
	}

	r.task = providerSubtask

	switch {
	case entry.UseIdleWatcher(r):
		wakerTask := providerSubtask.Parent().Subtask("waker for " + string(r.Alias))
		waker, err := idlewatcher.NewHTTPWaker(wakerTask, r.ReverseProxyEntry, r.rp)
		if err != nil {
			return err
		}
		r.handler = waker
		r.HealthMon = waker
	case entry.UseHealthCheck(r):
		r.HealthMon = monitor.NewHTTPHealthMonitor(r.rp.TargetURL, r.HealthCheck)
	}

	if r.handler == nil {
		switch {
		case len(r.PathPatterns) == 0:
			r.handler = r.rp
		case len(r.PathPatterns) == 1 && r.PathPatterns[0] == "/":
			r.handler = r.rp
		default:
			mux := gphttp.NewServeMux()
			patErrs := E.NewBuilder("invalid path pattern(s)")
			for _, p := range r.PathPatterns {
				patErrs.Add(mux.HandleFunc(p, r.rp.HandlerFunc))
			}
			if err := patErrs.Error(); err != nil {
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
		routes.SetHTTPRoute(string(r.Alias), r)
		r.task.OnFinished("remove from route table", func() {
			routes.DeleteHTTPRoute(string(r.Alias))
		})
	}

	if common.PrometheusEnabled {
		r.task.OnFinished("unreg metrics", r.rp.UnregisterMetrics)
	}
	return nil
}

// Finish implements task.TaskFinisher.
func (r *HTTPRoute) Finish(reason any) {
	r.task.Finish(reason)
}

func (r *HTTPRoute) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *HTTPRoute) addToLoadBalancer() {
	var lb *loadbalancer.LoadBalancer
	l, ok := routes.GetHTTPRoute(r.LoadBalance.Link)
	var linked *HTTPRoute
	if ok {
		linked = l.(*HTTPRoute)
		lb = linked.loadBalancer
		lb.UpdateConfigIfNeeded(r.LoadBalance)
		if linked.Raw.Homepage == nil && r.Raw.Homepage != nil {
			linked.Raw.Homepage = r.Raw.Homepage
		}
	} else {
		lb = loadbalancer.New(r.LoadBalance)
		lbTask := r.task.Parent().Subtask("loadbalancer " + r.LoadBalance.Link)
		lbTask.OnCancel("remove lb from routes", func() {
			routes.DeleteHTTPRoute(r.LoadBalance.Link)
		})
		lb.Start(lbTask)
		linked = &HTTPRoute{
			ReverseProxyEntry: &entry.ReverseProxyEntry{
				Raw: &route.RawEntry{
					Homepage: r.Raw.Homepage,
				},
				Alias: route.Alias(lb.Link),
			},
			HealthMon:    lb,
			loadBalancer: lb,
			handler:      lb,
		}
		routes.SetHTTPRoute(r.LoadBalance.Link, linked)
	}
	r.loadBalancer = lb
	r.server = loadbalance.NewServer(r.task.String(), r.rp.TargetURL, r.LoadBalance.Weight, r.handler, r.HealthMon)
	lb.AddServer(r.server)
	r.task.OnCancel("remove server from lb", func() {
		lb.RemoveServer(r.server)
	})
}
