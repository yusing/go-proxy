package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	T "github.com/yusing/go-proxy/proxy/fields"
)

type (
	ReverseProxyEntry struct { // real model after validation
		Alias        T.Alias
		Scheme       T.Scheme
		URL          *url.URL
		NoTLSVerify  bool
		PathPatterns T.PathPatterns
		SetHeaders   http.Header
		HideHeaders  []string

		/* Docker only */
		IdleTimeout      time.Duration
		WakeTimeout      time.Duration
		StopMethod       T.StopMethod
		StopTimeout      int
		StopSignal       T.Signal
		DockerHost       string
		ContainerName    string
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

func ValidateEntry(m *M.ProxyEntry) (any, E.NestedError) {
	m.SetDefaults()
	scheme, err := T.NewScheme(m.Scheme)
	if err.HasError() {
		return nil, err
	}

	var entry any
	e := E.NewBuilder("error validating proxy entry")
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

func validateRPEntry(m *M.ProxyEntry, s T.Scheme, b E.Builder) *ReverseProxyEntry {
	var stopTimeOut time.Duration

	host, err := T.ValidateHost(m.Host)
	b.Add(err)

	port, err := T.ValidatePort(m.Port)
	b.Add(err)

	pathPatterns, err := T.ValidatePathPatterns(m.PathPatterns)
	b.Add(err)

	setHeaders, err := T.ValidateHTTPHeaders(m.SetHeaders)
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
		URL:              url,
		NoTLSVerify:      m.NoTLSVerify,
		PathPatterns:     pathPatterns,
		SetHeaders:       setHeaders,
		HideHeaders:      m.HideHeaders,
		IdleTimeout:      idleTimeout,
		WakeTimeout:      wakeTimeout,
		StopMethod:       stopMethod,
		StopTimeout:      int(stopTimeOut.Seconds()), // docker api takes integer seconds for timeout argument
		StopSignal:       stopSignal,
		DockerHost:       m.DockerHost,
		ContainerName:    m.ContainerName,
		ContainerRunning: m.Running,
	}
}

func validateStreamEntry(m *M.ProxyEntry, b E.Builder) *StreamEntry {
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
