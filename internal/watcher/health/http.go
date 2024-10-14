package health

import (
	"crypto/tls"
	"errors"
	"net/http"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/net/types"
)

type HTTPHealthMonitor struct {
	*monitor
	method string
	pinger *http.Client
}

func NewHTTPHealthMonitor(task common.Task, url types.URL, config HealthCheckConfig) HealthMonitor {
	mon := new(HTTPHealthMonitor)
	mon.monitor = newMonitor(task, url, &config, mon.checkHealth)
	mon.pinger = &http.Client{Timeout: config.Timeout}
	if config.UseGet {
		mon.method = http.MethodGet
	} else {
		mon.method = http.MethodHead
	}
	return mon
}

func (mon *HTTPHealthMonitor) checkHealth() (healthy bool, detail string, err error) {
	req, reqErr := http.NewRequestWithContext(
		mon.task.Context(),
		mon.method,
		mon.URL.String(),
		nil,
	)
	if reqErr != nil {
		err = reqErr
		return
	}
	req.Header.Set("Connection", "close")

	resp, respErr := mon.pinger.Do(req)
	if respErr == nil {
		resp.Body.Close()
	}

	switch {
	case respErr != nil:
		// treat tls error as healthy
		var tlsErr *tls.CertificateVerificationError
		if ok := errors.As(respErr, &tlsErr); !ok {
			detail = respErr.Error()
			return
		}
	case resp.StatusCode == http.StatusServiceUnavailable:
		detail = resp.Status
		return
	}

	healthy = true
	return
}
