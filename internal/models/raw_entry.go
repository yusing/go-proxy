package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	. "github.com/yusing/go-proxy/internal/common"
	D "github.com/yusing/go-proxy/internal/docker"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	RawEntry struct {
		// raw entry object before validation
		// loaded from docker labels or yaml file
		Alias        string           `yaml:"-" json:"-"`
		Scheme       string           `yaml:"scheme" json:"scheme"`
		Host         string           `yaml:"host" json:"host"`
		Port         string           `yaml:"port" json:"port"`
		NoTLSVerify  bool             `yaml:"no_tls_verify" json:"no_tls_verify"` // https proxy only
		PathPatterns []string         `yaml:"path_patterns" json:"path_patterns"` // http(s) proxy only
		Middlewares  D.NestedLabelMap `yaml:"middlewares" json:"middlewares"`

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

	lp, pp, extra := e.splitPorts()

	if port, ok := ServiceNamePortMapTCP[e.ImageName]; ok {
		if pp == "" {
			pp = strconv.Itoa(port)
		}
		if e.Scheme == "" {
			e.Scheme = "tcp"
		}
	} else if port, ok := ImageNamePortMap[e.ImageName]; ok {
		if pp == "" {
			pp = strconv.Itoa(port)
		}
		if e.Scheme == "" {
			e.Scheme = "http"
		}
	} else if pp == "" && e.Scheme == "https" {
		pp = "443"
	} else if pp == "" {
		if p, ok := F.FirstValueOf(e.PrivatePortMapping); ok {
			pp = fmt.Sprint(p.PrivatePort)
		} else {
			pp = "80"
		}
	}

	// replace private port with public port (if any)
	if isDocker && e.NetworkMode != "host" {
		if p, ok := e.PrivatePortMapping[pp]; ok {
			pp = fmt.Sprint(p.PublicPort)
		}
		if _, ok := e.PublicPortMapping[pp]; !ok { // port is not exposed, but specified
			// try to fallback to first public port
			if p, ok := F.FirstValueOf(e.PublicPortMapping); ok {
				pp = fmt.Sprint(p.PublicPort)
			} else if e.Running {
				// ignore only if it is NOT RUNNING
				// because stopped containers
				// will have empty port mapping got from docker
				logrus.Debugf("ignored port %s for %s", pp, e.ContainerName)
				return false
			}
		}
	}

	if e.Scheme == "" && isDocker {
		if p, ok := e.PublicPortMapping[pp]; ok && p.Type == "udp" {
			e.Scheme = "udp"
		}
	}

	if e.Scheme == "" {
		if lp != "" {
			e.Scheme = "tcp"
		} else if strings.HasSuffix(pp, "443") {
			e.Scheme = "https"
		} else if _, ok := WellKnownHTTPPorts[pp]; ok {
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

	e.Port = joinPorts(lp, pp, extra)

	return true
}

func (e *RawEntry) splitPorts() (lp string, pp string, extra string) {
	portSplit := strings.Split(e.Port, ":")
	if len(portSplit) == 1 {
		pp = portSplit[0]
	} else {
		lp = portSplit[0]
		pp = portSplit[1]
	}
	if len(portSplit) > 2 {
		extra = strings.Join(portSplit[2:], ":")
	}
	return
}

func joinPorts(lp string, pp string, extra string) string {
	s := make([]string, 0, 3)
	if lp != "" {
		s = append(s, lp)
	}
	if pp != "" {
		s = append(s, pp)
	}
	if extra != "" {
		s = append(s, extra)
	}
	return strings.Join(s, ":")
}
