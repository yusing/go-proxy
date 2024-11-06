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
	HTTPRouteMetricLabels struct {
		Service, Method, Host, Visitor, Path string
	}
)

var rm RouteMetrics

const (
	routerNamespace     = "router"
	routerHTTPSubsystem = "http"
)

func GetRouteMetrics() *RouteMetrics {
	return &rm
}

func (lbl HTTPRouteMetricLabels) toPromLabels() prometheus.Labels {
	return prometheus.Labels{
		"service": lbl.Service,
		"method":  lbl.Method,
		"host":    lbl.Host,
		"visitor": lbl.Visitor,
		"path":    lbl.Path,
	}
}

func init() {
	if !common.PrometheusEnabled {
		return
	}
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
			Help:      "How long it took to process the request" + partitionsHelp,
		}, lbls...),
	}
}
