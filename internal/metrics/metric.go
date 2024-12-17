package metrics

import "github.com/prometheus/client_golang/prometheus"

type (
	Counter struct {
		mv        *prometheus.CounterVec
		collector prometheus.Counter
	}
	Gauge struct {
		mv        *prometheus.GaugeVec
		collector prometheus.Gauge
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

func (c *Counter) With(l Labels) *Counter {
	return &Counter{mv: c.mv, collector: c.mv.With(l.toPromLabels())}
}

func (c *Counter) Delete(l Labels) {
	c.mv.Delete(l.toPromLabels())
}

func (c *Counter) Reset() {
	c.mv.Reset()
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

func (g *Gauge) With(l Labels) *Gauge {
	return &Gauge{mv: g.mv, collector: g.mv.With(l.toPromLabels())}
}

func (g *Gauge) Delete(l Labels) {
	g.mv.Delete(l.toPromLabels())
}

func (g *Gauge) Reset() {
	g.mv.Reset()
}
