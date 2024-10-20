package entry

import (
	"fmt"

	"github.com/yusing/go-proxy/internal/docker"
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/proxy/fields"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type StreamEntry struct {
	Raw *RawEntry `json:"raw"`

	Alias       fields.Alias              `json:"alias"`
	Scheme      fields.StreamScheme       `json:"scheme"`
	URL         net.URL                   `json:"url"`
	Host        fields.Host               `json:"host,omitempty"`
	Port        fields.StreamPort         `json:"port,omitempty"`
	HealthCheck *health.HealthCheckConfig `json:"healthcheck,omitempty"`

	/* Docker only */
	Idlewatcher *idlewatcher.Config `json:"idlewatcher,omitempty"`
}

func (s *StreamEntry) TargetName() string {
	return string(s.Alias)
}

func (s *StreamEntry) TargetURL() net.URL {
	return s.URL
}

func (s *StreamEntry) RawEntry() *RawEntry {
	return s.Raw
}

func (s *StreamEntry) LoadBalanceConfig() *loadbalancer.Config {
	// TODO: support stream load balance
	return nil
}

func (s *StreamEntry) HealthCheckConfig() *health.HealthCheckConfig {
	return s.HealthCheck
}

func (s *StreamEntry) IdlewatcherConfig() *idlewatcher.Config {
	return s.Idlewatcher
}

func validateStreamEntry(m *RawEntry, b E.Builder) *StreamEntry {
	cont := m.Container
	if cont == nil {
		cont = docker.DummyContainer
	}

	host, err := fields.ValidateHost(m.Host)
	b.Add(err)

	port, err := fields.ValidateStreamPort(m.Port)
	b.Add(err)

	scheme, err := fields.ValidateStreamScheme(m.Scheme)
	b.Add(err)

	url, err := E.Check(net.ParseURL(fmt.Sprintf("%s://%s:%d", scheme.ProxyScheme, m.Host, port.ProxyPort)))
	b.Add(err)

	idleWatcherCfg, err := idlewatcher.ValidateConfig(m.Container)
	b.Add(err)

	if b.HasError() {
		return nil
	}

	return &StreamEntry{
		Raw:         m,
		Alias:       fields.NewAlias(m.Alias),
		Scheme:      *scheme,
		URL:         url,
		Host:        host,
		Port:        port,
		HealthCheck: m.HealthCheck,
		Idlewatcher: idleWatcherCfg,
	}
}
