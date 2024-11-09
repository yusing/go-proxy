package metrics

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/yusing/go-proxy/internal/common"
)

type (
	RouteMetrics struct {
		HTTPReqTotal,
		HTTP2xx3xx,
		HTTP4xx,
		HTTP5xx *Counter
		HTTPReqElapsed *Gauge
	}

	ServiceMetrics struct {
		HealthStatus *Gauge
	}
)

var (
	rm RouteMetrics
	sm ServiceMetrics
)

const (
	routerNamespace     = "router"
	routerHTTPSubsystem = "http"

	serviceNamespace = "service"
)

func GetRouteMetrics() *RouteMetrics {
	return &rm
}

func GetServiceMetrics() *ServiceMetrics {
	return &sm
}

func (rm *RouteMetrics) UnregisterService(service string) {
	lbls := &HTTPRouteMetricLabels{Service: service}
	prometheus.Unregister(rm.HTTP2xx3xx.With(lbls))
	prometheus.Unregister(rm.HTTP4xx.With(lbls))
	prometheus.Unregister(rm.HTTP5xx.With(lbls))
	prometheus.Unregister(rm.HTTPReqElapsed.With(lbls))
}

func init() {
	if !common.PrometheusEnabled {
		return
	}
	initRouteMetrics()
	initServiceMetrics()
}

func initRouteMetrics() {
	lbls := []string{"service", "method", "host", "visitor", "path"}
	partitionsHelp := ", partitioned by " + strings.Join(lbls, ", ")
	rm = RouteMetrics{
		HTTPReqTotal: NewCounter(prometheus.CounterOpts{
			Namespace: routerNamespace,
			Subsystem: routerHTTPSubsystem,
			Name:      "req_total",
			Help:      "How many requests processed" + partitionsHelp,
		}),
		HTTP2xx3xx: NewCounter(prometheus.CounterOpts{
			Namespace: routerNamespace,
			Subsystem: routerHTTPSubsystem,
			Name:      "req_ok_count",
			Help:      "How many 2xx-3xx requests processed" + partitionsHelp,
		}, lbls...),
		HTTP4xx: NewCounter(prometheus.CounterOpts{
			Namespace: routerNamespace,
			Subsystem: routerHTTPSubsystem,
			Name:      "req_4xx_count",
			Help:      "How many 4xx requests processed" + partitionsHelp,
		}, lbls...),
		HTTP5xx: NewCounter(prometheus.CounterOpts{
			Namespace: routerNamespace,
			Subsystem: routerHTTPSubsystem,
			Name:      "req_5xx_count",
			Help:      "How many 5xx requests processed" + partitionsHelp,
		}, lbls...),
		HTTPReqElapsed: NewGauge(prometheus.GaugeOpts{
			Namespace: routerNamespace,
			Subsystem: routerHTTPSubsystem,
			Name:      "req_elapsed_ms",
			Help:      "How long it took to process the request and respond a status code" + partitionsHelp,
		}, lbls...),
	}
}

func initServiceMetrics() {
	sm = ServiceMetrics{
		HealthStatus: NewGauge(prometheus.GaugeOpts{
			Namespace: serviceNamespace,
			Name:      "health_status",
			Help:      "The health status of the router by service",
		}, "service"),
	}
}
