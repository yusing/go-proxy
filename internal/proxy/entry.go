package proxy

import (
	"fmt"
	"net/url"
	"time"

	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	net "github.com/yusing/go-proxy/internal/net/types"
	T "github.com/yusing/go-proxy/internal/proxy/fields"
	"github.com/yusing/go-proxy/internal/types"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	ReverseProxyEntry struct { // real model after validation
		Raw *types.RawEntry `json:"raw"`

		Alias        T.Alias                   `json:"alias,omitempty"`
		Scheme       T.Scheme                  `json:"scheme,omitempty"`
		URL          net.URL                   `json:"url,omitempty"`
		NoTLSVerify  bool                      `json:"no_tls_verify,omitempty"`
		PathPatterns T.PathPatterns            `json:"path_patterns,omitempty"`
		HealthCheck  *health.HealthCheckConfig `json:"healthcheck,omitempty"`
		LoadBalance  *loadbalancer.Config      `json:"load_balance,omitempty"`
		Middlewares  D.NestedLabelMap          `json:"middlewares,omitempty"`

		/* Docker only */
		IdleTimeout      time.Duration `json:"idle_timeout,omitempty"`
		WakeTimeout      time.Duration `json:"wake_timeout,omitempty"`
		StopMethod       T.StopMethod  `json:"stop_method,omitempty"`
		StopTimeout      int           `json:"stop_timeout,omitempty"`
		StopSignal       T.Signal      `json:"stop_signal,omitempty"`
		DockerHost       string        `json:"docker_host,omitempty"`
		ContainerName    string        `json:"container_name,omitempty"`
		ContainerID      string        `json:"container_id,omitempty"`
		ContainerRunning bool          `json:"container_running,omitempty"`
	}
	StreamEntry struct {
		Raw *types.RawEntry `json:"raw"`

		Alias       T.Alias                   `json:"alias,omitempty"`
		Scheme      T.StreamScheme            `json:"scheme,omitempty"`
		Host        T.Host                    `json:"host,omitempty"`
		Port        T.StreamPort              `json:"port,omitempty"`
		Healthcheck *health.HealthCheckConfig `json:"healthcheck,omitempty"`
	}
)

func (rp *ReverseProxyEntry) UseIdleWatcher() bool {
	return rp.IdleTimeout > 0 && rp.IsDocker()
}

func (rp *ReverseProxyEntry) UseLoadBalance() bool {
	return rp.LoadBalance.Link != ""
}

func (rp *ReverseProxyEntry) IsDocker() bool {
	return rp.DockerHost != ""
}

func (rp *ReverseProxyEntry) IsZeroPort() bool {
	return rp.URL.Port() == "0"
}

func (rp *ReverseProxyEntry) ShouldNotServe() bool {
	return rp.IsZeroPort() && !rp.UseIdleWatcher()
}

func ValidateEntry(m *types.RawEntry) (any, E.NestedError) {
	m.FillMissingFields()

	scheme, err := T.NewScheme(m.Scheme)
	if err != nil {
		return nil, err
	}

	var entry any
	e := E.NewBuilder("error validating entry")
	if scheme.IsStream() {
		entry = validateStreamEntry(m, e)
	} else {
		entry = validateRPEntry(m, scheme, e)
	}
	if err := e.Build(); err != nil {
		return nil, err
	}
	return entry, nil
}

func validateRPEntry(m *types.RawEntry, s T.Scheme, b E.Builder) *ReverseProxyEntry {
	var stopTimeOut time.Duration
	cont := m.Container
	if cont == nil {
		cont = D.DummyContainer
	}

	host, err := T.ValidateHost(m.Host)
	b.Add(err)

	port, err := T.ValidatePort(m.Port)
	b.Add(err)

	pathPatterns, err := T.ValidatePathPatterns(m.PathPatterns)
	b.Add(err)

	url, err := E.Check(url.Parse(fmt.Sprintf("%s://%s:%d", s, host, port)))
	b.Add(err)

	idleTimeout, err := T.ValidateDurationPostitive(cont.IdleTimeout)
	b.Add(err)

	wakeTimeout, err := T.ValidateDurationPostitive(cont.WakeTimeout)
	b.Add(err)

	stopMethod, err := T.ValidateStopMethod(cont.StopMethod)
	b.Add(err)

	if stopMethod == T.StopMethodStop {
		stopTimeOut, err = T.ValidateDurationPostitive(cont.StopTimeout)
		b.Add(err)
	}

	stopSignal, err := T.ValidateSignal(cont.StopSignal)
	b.Add(err)

	if err != nil {
		return nil
	}

	return &ReverseProxyEntry{
		Raw:              m,
		Alias:            T.NewAlias(m.Alias),
		Scheme:           s,
		URL:              net.NewURL(url),
		NoTLSVerify:      m.NoTLSVerify,
		PathPatterns:     pathPatterns,
		HealthCheck:      &m.HealthCheck,
		LoadBalance:      &m.LoadBalance,
		Middlewares:      m.Middlewares,
		IdleTimeout:      idleTimeout,
		WakeTimeout:      wakeTimeout,
		StopMethod:       stopMethod,
		StopTimeout:      int(stopTimeOut.Seconds()), // docker api takes integer seconds for timeout argument
		StopSignal:       stopSignal,
		DockerHost:       cont.DockerHost,
		ContainerName:    cont.ContainerName,
		ContainerID:      cont.ContainerID,
		ContainerRunning: cont.Running,
	}
}

func validateStreamEntry(m *types.RawEntry, b E.Builder) *StreamEntry {
	host, err := T.ValidateHost(m.Host)
	b.Add(err)

	port, err := T.ValidateStreamPort(m.Port)
	b.Add(err)

	scheme, err := T.ValidateStreamScheme(m.Scheme)
	b.Add(err)

	if b.HasError() {
		return nil
	}

	return &StreamEntry{
		Raw:         m,
		Alias:       T.NewAlias(m.Alias),
		Scheme:      *scheme,
		Host:        host,
		Port:        port,
		Healthcheck: &m.HealthCheck,
	}
}
