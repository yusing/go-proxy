package model

import (
	"strconv"
	"strings"

	. "github.com/yusing/go-proxy/common"
	D "github.com/yusing/go-proxy/docker"
	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	ProxyEntry struct { // raw entry object before validation
		Alias        string            `yaml:"-" json:"-"`
		Scheme       string            `yaml:"scheme" json:"scheme"`
		Host         string            `yaml:"host" json:"host"`
		Port         string            `yaml:"port" json:"port"`
		NoTLSVerify  bool              `yaml:"no_tls_verify" json:"no_tls_verify"` // https proxy only
		PathPatterns []string          `yaml:"path_patterns" json:"path_patterns"` // http(s) proxy only
		SetHeaders   map[string]string `yaml:"set_headers" json:"set_headers"`     // http(s) proxy only
		HideHeaders  []string          `yaml:"hide_headers" json:"hide_headers"`   // http(s) proxy only

		/* Docker only */
		*D.ProxyProperties `yaml:"-" json:"proxy_properties"`
	}

	ProxyEntries = F.Map[string, *ProxyEntry]
)

var NewProxyEntries = F.NewMapOf[string, *ProxyEntry]

func (e *ProxyEntry) SetDefaults() {
	if e.ProxyProperties == nil {
		e.ProxyProperties = &D.ProxyProperties{}
	}

	if e.Scheme == "" {
		switch {
		case strings.ContainsRune(e.Port, ':'):
			e.Scheme = "tcp"
		case e.ProxyProperties != nil:
			if _, ok := ServiceNamePortMapTCP[e.ImageName]; ok {
				e.Scheme = "tcp"
			}
		}
	}

	if e.Scheme == "" {
		switch e.Port {
		case "443", "8443":
			e.Scheme = "https"
		default:
			e.Scheme = "http"
		}
	}
	if e.Host == "" {
		e.Host = "localhost"
	}
	if e.Port == "" {
		e.Port = e.FirstPort
	}
	if e.Port == "" {
		if port, ok := ServiceNamePortMapTCP[e.Port]; ok {
			e.Port = strconv.Itoa(port)
		} else if port, ok := ImageNamePortMapHTTP[e.Port]; ok {
			e.Port = strconv.Itoa(port)
		} else {
			switch e.Scheme {
			case "http":
				e.Port = "80"
			case "https":
				e.Port = "443"
			}
		}
	}
	if e.IdleTimeout == "" {
		e.IdleTimeout = IdleTimeoutDefault
	}
	if e.WakeTimeout == "" {
		e.WakeTimeout = WakeTimeoutDefault
	}
	if e.StopTimeout == "" {
		e.StopTimeout = StopTimeoutDefault
	}
	if e.StopMethod == "" {
		e.StopMethod = StopMethodDefault
	}
}
