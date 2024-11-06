package metrics

import "github.com/prometheus/client_golang/prometheus"

func InitRouterMetrics(getRPsCount func() int, getStreamsCount func() int) {
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "entrypoint",
		Name:      "num_reverse_proxies",
		Help:      "The number of reverse proxies",
	}, func() float64 {
		return float64(getRPsCount())
	}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "entrypoint",
		Name:      "num_streams",
		Help:      "The number of streams",
	}, func() float64 {
		return float64(getStreamsCount())
	}))
}
