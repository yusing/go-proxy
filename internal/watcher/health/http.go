package health

import (
	"crypto/tls"
	"errors"
	"net/http"

	"github.com/yusing/go-proxy/internal/net/types"
)

type HTTPHealthMonitor struct {
	*monitor
	method string
	pinger *http.Client
}

func NewHTTPHealthMonitor(url types.URL, config *HealthCheckConfig, transport http.RoundTripper) *HTTPHealthMonitor {
	mon := new(HTTPHealthMonitor)
	mon.monitor = newMonitor(url, config, mon.CheckHealth)
	mon.pinger = &http.Client{Timeout: config.Timeout, Transport: transport}
	if config.UseGet {
		mon.method = http.MethodGet
	} else {
		mon.method = http.MethodHead
	}
	return mon
}

func NewHTTPHealthChecker(url types.URL, config *HealthCheckConfig, transport http.RoundTripper) HealthChecker {
	return NewHTTPHealthMonitor(url, config, transport)
}

func (mon *HTTPHealthMonitor) CheckHealth() (healthy bool, detail string, err error) {
	ctx, cancel := mon.ContextWithTimeout("ping request timed out")
	defer cancel()

	req, reqErr := http.NewRequestWithContext(
		ctx,
		mon.method,
		mon.url.Load().JoinPath(mon.config.Path).String(),
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
