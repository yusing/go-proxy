package model

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/yusing/go-proxy/common"
	D "github.com/yusing/go-proxy/docker"
	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	RawEntry struct {
		// raw entry object before validation
		// loaded from docker labels or yaml file
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

	RawEntries = F.Map[string, *RawEntry]
)

var NewProxyEntries = F.NewMapOf[string, *RawEntry]

func (e *RawEntry) FillMissingFields() bool {
	isDocker := e.ProxyProperties != nil

	if !isDocker {
		e.ProxyProperties = &D.ProxyProperties{}
	}

	if e.Port == "" {
		if port, ok := ServiceNamePortMapTCP[e.ImageName]; ok {
			e.Port = strconv.Itoa(port)
		} else if port, ok := ImageNamePortMap[e.ImageName]; ok {
			e.Port = strconv.Itoa(port)
		} else {
			switch {
			case e.Scheme == "https":
				e.Port = "443"
			case !isDocker:
				e.Port = "80"
			}
		}
	}

	if e.PublicPortMapping != nil {
		if _, ok := e.PublicPortMapping[e.Port]; !ok { // port is not exposed, but specified
			// try to fallback to first public port
			if len(e.PublicPortMapping) == 0 {
				return false
			}
			for _, p := range e.PublicPortMapping {
				e.Port = fmt.Sprint(p.PublicPort)
				break
			}
		}
	}

	if e.Scheme == "" {
		if _, ok := ServiceNamePortMapTCP[e.ImageName]; ok {
			e.Scheme = "tcp"
		} else if strings.ContainsRune(e.Port, ':') {
			e.Scheme = "tcp"
		} else if _, ok := WellKnownHTTPPorts[e.Port]; ok {
			e.Scheme = "http"
		} else if e.Port == "443" {
			e.Scheme = "https"
		} else if isDocker {
			if e.Port == "" {
				return false
			}
			if p, ok := e.PublicPortMapping[e.Port]; ok {
				if p.Type == "udp" {
					e.Scheme = "udp"
				} else {
					e.Scheme = "http"
				}
			} else {
				return false
			}
		} else {
			e.Scheme = "http"
		}
	}

	if e.Host == "" {
		e.Host = "localhost"
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

	return true
}
