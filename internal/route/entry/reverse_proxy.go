package entry

import (
	"fmt"
	"net/url"

	"github.com/yusing/go-proxy/internal/docker"
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	loadbalance "github.com/yusing/go-proxy/internal/net/http/loadbalancer/types"
	net "github.com/yusing/go-proxy/internal/net/types"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type ReverseProxyEntry struct { // real model after validation
	Raw *route.RawEntry `json:"raw"`

	Alias        route.Alias                `json:"alias"`
	Scheme       route.Scheme               `json:"scheme"`
	URL          net.URL                    `json:"url"`
	NoTLSVerify  bool                       `json:"no_tls_verify,omitempty"`
	PathPatterns route.PathPatterns         `json:"path_patterns,omitempty"`
	HealthCheck  *health.HealthCheckConfig  `json:"healthcheck,omitempty"`
	LoadBalance  *loadbalance.Config        `json:"load_balance,omitempty"`
	Middlewares  map[string]docker.LabelMap `json:"middlewares,omitempty"`

	/* Docker only */
	Idlewatcher *idlewatcher.Config `json:"idlewatcher,omitempty"`
}

func (rp *ReverseProxyEntry) TargetName() string {
	return string(rp.Alias)
}

func (rp *ReverseProxyEntry) TargetURL() net.URL {
	return rp.URL
}

func (rp *ReverseProxyEntry) RawEntry() *route.RawEntry {
	return rp.Raw
}

func (rp *ReverseProxyEntry) LoadBalanceConfig() *loadbalance.Config {
	return rp.LoadBalance
}

func (rp *ReverseProxyEntry) HealthCheckConfig() *health.HealthCheckConfig {
	return rp.HealthCheck
}

func (rp *ReverseProxyEntry) IdlewatcherConfig() *idlewatcher.Config {
	return rp.Idlewatcher
}

func validateRPEntry(m *route.RawEntry, s route.Scheme, errs *E.Builder) *ReverseProxyEntry {
	cont := m.Container
	if cont == nil {
		cont = docker.DummyContainer
	}

	lb := m.LoadBalance
	if lb != nil && lb.Link == "" {
		lb = nil
	}

	host := E.Collect(errs, route.ValidateHost, m.Host)
	port := E.Collect(errs, route.ValidatePort, m.Port)
	pathPats := E.Collect(errs, route.ValidatePathPatterns, m.PathPatterns)
	url := E.Collect(errs, url.Parse, fmt.Sprintf("%s://%s:%d", s, host, port))
	iwCfg := E.Collect(errs, idlewatcher.ValidateConfig, cont)

	if errs.HasError() {
		return nil
	}

	return &ReverseProxyEntry{
		Raw:          m,
		Alias:        route.Alias(m.Alias),
		Scheme:       s,
		URL:          net.NewURL(url),
		NoTLSVerify:  m.NoTLSVerify,
		PathPatterns: pathPats,
		HealthCheck:  m.HealthCheck,
		LoadBalance:  lb,
		Middlewares:  m.Middlewares,
		Idlewatcher:  iwCfg,
	}
}
