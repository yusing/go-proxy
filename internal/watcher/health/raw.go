package health

import (
	"net"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/net/types"
)

type (
	RawHealthMonitor struct {
		*monitor
		dialer *net.Dialer
	}
)

func NewRawHealthMonitor(task common.Task, url types.URL, config HealthCheckConfig) HealthMonitor {
	mon := new(RawHealthMonitor)
	mon.monitor = newMonitor(task, url, &config, mon.checkAvail)
	mon.dialer = &net.Dialer{
		Timeout:       config.Timeout,
		FallbackDelay: -1,
	}
	return mon
}

func (mon *RawHealthMonitor) checkAvail() (avail bool, detail string, err error) {
	conn, dialErr := mon.dialer.DialContext(mon.task.Context(), mon.URL.Scheme, mon.URL.Host)
	if dialErr != nil {
		detail = dialErr.Error()
		/* trunk-ignore(golangci-lint/nilerr) */
		return
	}
	conn.Close()
	avail = true
	return
}
