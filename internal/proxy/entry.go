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
)

type (
	ReverseProxyEntry struct { // real model after validation
		Alias        T.Alias
		Scheme       T.Scheme
		URL          net.URL
		NoTLSVerify  bool
		PathPatterns T.PathPatterns
		LoadBalance  loadbalancer.Config
		Middlewares  D.NestedLabelMap

		/* Docker only */
		IdleTimeout      time.Duration
		WakeTimeout      time.Duration
		StopMethod       T.StopMethod
		StopTimeout      int
		StopSignal       T.Signal
		DockerHost       string
		ContainerName    string
		ContainerID      string
		ContainerRunning bool
	}
	StreamEntry struct {
		Alias  T.Alias        `json:"alias"`
		Scheme T.StreamScheme `json:"scheme"`
		Host   T.Host         `json:"host"`
		Port   T.StreamPort   `json:"port"`
	}
)

func (rp *ReverseProxyEntry) UseIdleWatcher() bool {
	return rp.IdleTimeout > 0 && rp.DockerHost != ""
}

func (rp *ReverseProxyEntry) IsDocker() bool {
	return rp.DockerHost != ""
}

func (rp *ReverseProxyEntry) IsZeroPort() bool {
	return rp.URL.Port() == "0"
}

func ValidateEntry(m *types.RawEntry) (any, E.NestedError) {
	m.FillMissingFields()

	scheme, err := T.NewScheme(m.Scheme)
	if err.HasError() {
		return nil, err
	}

	var entry any
	e := E.NewBuilder("error validating entry")
	if scheme.IsStream() {
		entry = validateStreamEntry(m, e)
	} else {
		entry = validateRPEntry(m, scheme, e)
	}
	if err := e.Build(); err.HasError() {
		return nil, err
	}
	return entry, nil
}

func validateRPEntry(m *types.RawEntry, s T.Scheme, b E.Builder) *ReverseProxyEntry {
	var stopTimeOut time.Duration

	host, err := T.ValidateHost(m.Host)
	b.Add(err)

	port, err := T.ValidatePort(m.Port)
	b.Add(err)

	pathPatterns, err := T.ValidatePathPatterns(m.PathPatterns)
	b.Add(err)

	url, err := E.Check(url.Parse(fmt.Sprintf("%s://%s:%d", s, host, port)))
	b.Add(err)

	idleTimeout, err := T.ValidateDurationPostitive(m.IdleTimeout)
	b.Add(err)

	wakeTimeout, err := T.ValidateDurationPostitive(m.WakeTimeout)
	b.Add(err)

	stopMethod, err := T.ValidateStopMethod(m.StopMethod)
	b.Add(err)

	if stopMethod == T.StopMethodStop {
		stopTimeOut, err = T.ValidateDurationPostitive(m.StopTimeout)
		b.Add(err)
	}

	stopSignal, err := T.ValidateSignal(m.StopSignal)
	b.Add(err)

	if err.HasError() {
		return nil
	}

	return &ReverseProxyEntry{
		Alias:            T.NewAlias(m.Alias),
		Scheme:           s,
		URL:              net.NewURL(url),
		NoTLSVerify:      m.NoTLSVerify,
		PathPatterns:     pathPatterns,
		LoadBalance:      m.LoadBalance,
		Middlewares:      m.Middlewares,
		IdleTimeout:      idleTimeout,
		WakeTimeout:      wakeTimeout,
		StopMethod:       stopMethod,
		StopTimeout:      int(stopTimeOut.Seconds()), // docker api takes integer seconds for timeout argument
		StopSignal:       stopSignal,
		DockerHost:       m.DockerHost,
		ContainerName:    m.ContainerName,
		ContainerID:      m.ContainerID,
		ContainerRunning: m.Running,
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
		Alias:  T.NewAlias(m.Alias),
		Scheme: *scheme,
		Host:   host,
		Port:   port,
	}
}
