package types

import (
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	RawEntry struct {
		_ U.NoCopy

		// raw entry object before validation
		// loaded from docker labels or yaml file
		Alias        string                   `json:"-" yaml:"-"`
		Scheme       string                   `json:"scheme" yaml:"scheme"`
		Host         string                   `json:"host" yaml:"host"`
		Port         string                   `json:"port" yaml:"port"`
		NoTLSVerify  bool                     `json:"no_tls_verify,omitempty" yaml:"no_tls_verify"` // https proxy only
		PathPatterns []string                 `json:"path_patterns,omitempty" yaml:"path_patterns"` // http(s) proxy only
		HealthCheck  health.HealthCheckConfig `json:"healthcheck,omitempty" yaml:"healthcheck"`
		LoadBalance  loadbalancer.Config      `json:"load_balance,omitempty" yaml:"load_balance"`
		Middlewares  docker.NestedLabelMap    `json:"middlewares,omitempty" yaml:"middlewares"`
		Homepage     *homepage.Item           `json:"homepage,omitempty" yaml:"homepage"`

		/* Docker only */
		*docker.Container `json:"container" yaml:"-"`
	}

	RawEntries = F.Map[string, *RawEntry]
)

var NewProxyEntries = F.NewMapOf[string, *RawEntry]

func (e *RawEntry) FillMissingFields() {
	isDocker := e.Container != nil
	if !isDocker {
		e.Container = &docker.Container{}
	}

	if e.Host == "" {
		switch {
		case e.PrivateIP != "":
			e.Host = e.PrivateIP
		case e.PublicIP != "":
			e.Host = e.PublicIP
		case !isDocker:
			e.Host = "localhost"
		}
	}

	lp, pp, extra := e.splitPorts()

	if port, ok := common.ServiceNamePortMapTCP[e.ImageName]; ok {
		if pp == "" {
			pp = strconv.Itoa(port)
		}
		if e.Scheme == "" {
			e.Scheme = "tcp"
		}
	} else if port, ok := common.ImageNamePortMap[e.ImageName]; ok {
		if pp == "" {
			pp = strconv.Itoa(port)
		}
		if e.Scheme == "" {
			e.Scheme = "http"
		}
	} else if pp == "" && e.Scheme == "https" {
		pp = "443"
	} else if pp == "" {
		if p := lowestPort(e.PrivatePortMapping); p != "" {
			pp = p
		} else if p := lowestPort(e.PublicPortMapping); p != "" {
			pp = p
		} else if !isDocker {
			pp = "80"
		} else {
			logrus.Debugf("no port found for %s", e.Alias)
		}
	}

	// replace private port with public port if using public IP.
	if e.Host == e.PublicIP {
		if p, ok := e.PrivatePortMapping[pp]; ok {
			pp = U.PortString(p.PublicPort)
		}
	}
	// replace public port with private port if using private IP.
	if e.Host == e.PrivateIP {
		if p, ok := e.PublicPortMapping[pp]; ok {
			pp = U.PortString(p.PrivatePort)
		}
	}

	if e.Scheme == "" && isDocker {
		switch {
		case e.Host == e.PublicIP && e.PublicPortMapping[pp].Type == "udp":
			e.Scheme = "udp"
		case e.Host == e.PrivateIP && e.PrivatePortMapping[pp].Type == "udp":
			e.Scheme = "udp"
		}
	}

	if e.Scheme == "" {
		switch {
		case lp != "":
			e.Scheme = "tcp"
		case strings.HasSuffix(pp, "443"):
			e.Scheme = "https"
		default: // assume its http
			e.Scheme = "http"
		}
	}

	if e.HealthCheck.Interval == 0 {
		e.HealthCheck.Interval = common.HealthCheckIntervalDefault
	}
	if e.HealthCheck.Timeout == 0 {
		e.HealthCheck.Timeout = common.HealthCheckTimeoutDefault
	}
	if e.IdleTimeout == "" {
		e.IdleTimeout = common.IdleTimeoutDefault
	}
	if e.WakeTimeout == "" {
		e.WakeTimeout = common.WakeTimeoutDefault
	}
	if e.StopTimeout == "" {
		e.StopTimeout = common.StopTimeoutDefault
	}
	if e.StopMethod == "" {
		e.StopMethod = common.StopMethodDefault
	}

	e.Port = joinPorts(lp, pp, extra)

	if e.Port == "" || e.Host == "" {
		e.Port = "0"
	}
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

func lowestPort(ports map[string]types.Port) string {
	var cmp uint16
	var res string
	for port, v := range ports {
		if v.PrivatePort < cmp || cmp == 0 {
			cmp = v.PrivatePort
			res = port
		}
	}
	return res
}
