package health

import (
	"crypto/tls"
	"errors"
	"net/http"

	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/pkg"
)

type HTTPHealthMonitor struct {
	*monitor
	method string
}

var pinger = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		ForceAttemptHTTP2: false,
	},
	CheckRedirect: func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func NewHTTPHealthMonitor(url types.URL, config *HealthCheckConfig) *HTTPHealthMonitor {
	mon := new(HTTPHealthMonitor)
	mon.monitor = newMonitor(url, config, mon.CheckHealth)
	if config.UseGet {
		mon.method = http.MethodGet
	} else {
		mon.method = http.MethodHead
	}
	return mon
}

func NewHTTPHealthChecker(url types.URL, config *HealthCheckConfig) HealthChecker {
	return NewHTTPHealthMonitor(url, config)
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
	req.Header.Set("User-Agent", "GoDoxy/"+pkg.GetVersion())
	resp, respErr := pinger.Do(req)
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
