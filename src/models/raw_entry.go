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

	if port, ok := ServiceNamePortMapTCP[e.ImageName]; ok {
		e.Port = strconv.Itoa(port)
		e.Scheme = "tcp"
	} else if port, ok := ImageNamePortMap[e.ImageName]; ok {
		e.Port = strconv.Itoa(port)
		e.Scheme = "http"
	} else if e.Port == "" && e.Scheme == "https" {
		e.Port = "443"
	} else if e.Port == "" {
		e.Port = "80"
	}

	// replace private port with public port (if any)
	if isDocker && e.NetworkMode != "host" {
		if _, ok := e.PublicPortMapping[e.Port]; !ok { // port is not exposed, but specified
			// try to fallback to first public port
			if p, ok := F.FirstValueOf(e.PublicPortMapping); ok {
				e.Port = fmt.Sprint(p.PublicPort)
			}
			// ignore only if it is NOT RUNNING
			// because stopped containers
			// will have empty port mapping got from docker
			if e.Running {
				return false
			}
		}
	}

	if e.Scheme == "" && isDocker {
		if p, ok := e.PublicPortMapping[e.Port]; ok {
			if p.Type == "udp" {
				e.Scheme = "udp"
			} else {
				e.Scheme = "http"
			}
		}
	}

	if e.Scheme == "" {
		if strings.ContainsRune(e.Port, ':') {
			e.Scheme = "tcp"
		} else if strings.HasSuffix(e.Port, "443") {
			e.Scheme = "https"
		} else if _, ok := WellKnownHTTPPorts[e.Port]; ok {
			e.Scheme = "http"
		} else {
			// assume its http
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
