package metrics

import "github.com/prometheus/client_golang/prometheus"

type (
	Counter struct {
		collector prometheus.Counter
		mv        *prometheus.CounterVec
	}
	Gauge struct {
		collector prometheus.Gauge
		mv        *prometheus.GaugeVec
	}
	Labels interface {
		toPromLabels() prometheus.Labels
	}
)

func NewCounter(opts prometheus.CounterOpts, labels ...string) *Counter {
	m := &Counter{
		mv: prometheus.NewCounterVec(opts, labels),
	}
	if len(labels) == 0 {
		m.collector = m.mv.WithLabelValues()
		m.collector.Add(0)
	}
	prometheus.MustRegister(m)
	return m
}

func NewGauge(opts prometheus.GaugeOpts, labels ...string) *Gauge {
	m := &Gauge{
		mv: prometheus.NewGaugeVec(opts, labels),
	}
	if len(labels) == 0 {
		m.collector = m.mv.WithLabelValues()
		m.collector.Set(0)
	}
	prometheus.MustRegister(m)
	return m
}

func (c *Counter) Collect(ch chan<- prometheus.Metric) {
	c.mv.Collect(ch)
}

func (c *Counter) Describe(ch chan<- *prometheus.Desc) {
	c.mv.Describe(ch)
}

func (c *Counter) Inc() {
	c.collector.Inc()
}

func (c *Counter) With(l Labels) prometheus.Counter {
	return c.mv.With(l.toPromLabels())
}

func (g *Gauge) Collect(ch chan<- prometheus.Metric) {
	g.mv.Collect(ch)
}

func (g *Gauge) Describe(ch chan<- *prometheus.Desc) {
	g.mv.Describe(ch)
}

func (g *Gauge) Set(v float64) {
	g.collector.Set(v)
}

func (g *Gauge) With(l Labels) prometheus.Gauge {
	return g.mv.With(l.toPromLabels())
}
