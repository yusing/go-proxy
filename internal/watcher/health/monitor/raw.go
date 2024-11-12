package monitor

import (
	"net"

	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	RawHealthMonitor struct {
		*monitor
		dialer *net.Dialer
	}
)

func NewRawHealthMonitor(url types.URL, config *health.HealthCheckConfig) *RawHealthMonitor {
	mon := new(RawHealthMonitor)
	mon.monitor = newMonitor(url, config, mon.CheckHealth)
	mon.dialer = &net.Dialer{
		Timeout:       config.Timeout,
		FallbackDelay: -1,
	}
	return mon
}

func NewRawHealthChecker(url types.URL, config *health.HealthCheckConfig) health.HealthChecker {
	return NewRawHealthMonitor(url, config)
}

func (mon *RawHealthMonitor) CheckHealth() (healthy bool, detail string, err error) {
	ctx, cancel := mon.ContextWithTimeout("ping request timed out")
	defer cancel()

	url := mon.url.Load()
	conn, dialErr := mon.dialer.DialContext(ctx, url.Scheme, url.Host)
	if dialErr != nil {
		detail = dialErr.Error()
		/* trunk-ignore(golangci-lint/nilerr) */
		return
	}
	conn.Close()
	healthy = true
	return
}
