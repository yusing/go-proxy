package entry

import (
	"fmt"
	"net/url"

	"github.com/yusing/go-proxy/internal/docker"
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/proxy/fields"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type ReverseProxyEntry struct { // real model after validation
	Raw *RawEntry `json:"raw"`

	Alias        fields.Alias              `json:"alias,omitempty"`
	Scheme       fields.Scheme             `json:"scheme,omitempty"`
	URL          net.URL                   `json:"url,omitempty"`
	NoTLSVerify  bool                      `json:"no_tls_verify,omitempty"`
	PathPatterns fields.PathPatterns       `json:"path_patterns,omitempty"`
	HealthCheck  *health.HealthCheckConfig `json:"healthcheck,omitempty"`
	LoadBalance  *loadbalancer.Config      `json:"load_balance,omitempty"`
	Middlewares  docker.NestedLabelMap     `json:"middlewares,omitempty"`

	/* Docker only */
	Idlewatcher *idlewatcher.Config `json:"idlewatcher,omitempty"`
}

func (rp *ReverseProxyEntry) TargetName() string {
	return string(rp.Alias)
}

func (rp *ReverseProxyEntry) TargetURL() net.URL {
	return rp.URL
}

func (rp *ReverseProxyEntry) RawEntry() *RawEntry {
	return rp.Raw
}

func (rp *ReverseProxyEntry) LoadBalanceConfig() *loadbalancer.Config {
	return rp.LoadBalance
}

func (rp *ReverseProxyEntry) HealthCheckConfig() *health.HealthCheckConfig {
	return rp.HealthCheck
}

func (rp *ReverseProxyEntry) IdlewatcherConfig() *idlewatcher.Config {
	return rp.Idlewatcher
}

func validateRPEntry(m *RawEntry, s fields.Scheme, b E.Builder) *ReverseProxyEntry {
	cont := m.Container
	if cont == nil {
		cont = docker.DummyContainer
	}

	lb := m.LoadBalance
	if lb != nil && lb.Link == "" {
		lb = nil
	}

	host, err := fields.ValidateHost(m.Host)
	b.Add(err)

	port, err := fields.ValidatePort(m.Port)
	b.Add(err)

	pathPatterns, err := fields.ValidatePathPatterns(m.PathPatterns)
	b.Add(err)

	url, err := E.Check(url.Parse(fmt.Sprintf("%s://%s:%d", s, host, port)))
	b.Add(err)

	idleWatcherCfg, err := idlewatcher.ValidateConfig(m.Container)
	b.Add(err)

	if err != nil {
		return nil
	}

	return &ReverseProxyEntry{
		Raw:          m,
		Alias:        fields.NewAlias(m.Alias),
		Scheme:       s,
		URL:          net.NewURL(url),
		NoTLSVerify:  m.NoTLSVerify,
		PathPatterns: pathPatterns,
		HealthCheck:  m.HealthCheck,
		LoadBalance:  lb,
		Middlewares:  m.Middlewares,
		Idlewatcher:  idleWatcherCfg,
	}
}
