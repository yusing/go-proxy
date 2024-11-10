package entry

import (
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	RawEntry struct {
		_ U.NoCopy

		// raw entry object before validation
		// loaded from docker labels or yaml file
		Alias        string                     `json:"-" yaml:"-"`
		Scheme       string                     `json:"scheme,omitempty" yaml:"scheme"`
		Host         string                     `json:"host,omitempty" yaml:"host"`
		Port         string                     `json:"port,omitempty" yaml:"port"`
		NoTLSVerify  bool                       `json:"no_tls_verify,omitempty" yaml:"no_tls_verify"` // https proxy only
		PathPatterns []string                   `json:"path_patterns,omitempty" yaml:"path_patterns"` // http(s) proxy only
		HealthCheck  *health.HealthCheckConfig  `json:"healthcheck,omitempty" yaml:"healthcheck"`
		LoadBalance  *loadbalancer.Config       `json:"load_balance,omitempty" yaml:"load_balance"`
		Middlewares  map[string]docker.LabelMap `json:"middlewares,omitempty" yaml:"middlewares"`
		Homepage     *homepage.Item             `json:"homepage,omitempty" yaml:"homepage"`

		/* Docker only */
		Container *docker.Container `json:"container,omitempty" yaml:"-"`

		finalized bool
	}

	RawEntries = F.Map[string, *RawEntry]
)

var NewProxyEntries = F.NewMapOf[string, *RawEntry]

func (e *RawEntry) Finalize() {
	if e.finalized {
		return
	}

	isDocker := e.Container != nil
	cont := e.Container
	if !isDocker {
		cont = docker.DummyContainer
	}

	if e.Host == "" {
		switch {
		case cont.PrivateIP != "":
			e.Host = cont.PrivateIP
		case cont.PublicIP != "":
			e.Host = cont.PublicIP
		case !isDocker:
			e.Host = "localhost"
		}
	}

	lp, pp, extra := e.splitPorts()

	if port, ok := common.ServiceNamePortMapTCP[cont.ImageName]; ok {
		if pp == "" {
			pp = strconv.Itoa(port)
		}
		if e.Scheme == "" {
			e.Scheme = "tcp"
		}
	} else if port, ok := common.ImageNamePortMap[cont.ImageName]; ok {
		if pp == "" {
			pp = strconv.Itoa(port)
		}
		if e.Scheme == "" {
			e.Scheme = "http"
		}
	} else if pp == "" && e.Scheme == "https" {
		pp = "443"
	} else if pp == "" {
		if p := lowestPort(cont.PrivatePortMapping); p != "" {
			pp = p
		} else if p := lowestPort(cont.PublicPortMapping); p != "" {
			pp = p
		} else if !isDocker {
			pp = "80"
		} else {
			logging.Debug().Msg("no port found for " + e.Alias)
		}
	}

	// replace private port with public port if using public IP.
	if e.Host == cont.PublicIP {
		if p, ok := cont.PrivatePortMapping[pp]; ok {
			pp = strutils.PortString(p.PublicPort)
		}
	}
	// replace public port with private port if using private IP.
	if e.Host == cont.PrivateIP {
		if p, ok := cont.PublicPortMapping[pp]; ok {
			pp = strutils.PortString(p.PrivatePort)
		}
	}

	if e.Scheme == "" && isDocker {
		switch {
		case e.Host == cont.PublicIP && cont.PublicPortMapping[pp].Type == "udp":
			e.Scheme = "udp"
		case e.Host == cont.PrivateIP && cont.PrivatePortMapping[pp].Type == "udp":
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

	if e.HealthCheck == nil {
		e.HealthCheck = health.DefaultHealthCheckConfig()
	}

	if e.HealthCheck.Disable {
		e.HealthCheck = nil
	}

	if cont.IdleTimeout != "" {
		if cont.WakeTimeout == "" {
			cont.WakeTimeout = common.WakeTimeoutDefault
		}
		if cont.StopTimeout == "" {
			cont.StopTimeout = common.StopTimeoutDefault
		}
		if cont.StopMethod == "" {
			cont.StopMethod = common.StopMethodDefault
		}
	}

	e.Port = joinPorts(lp, pp, extra)

	if e.Port == "" || e.Host == "" {
		if lp != "" {
			e.Port = lp + ":0"
		} else {
			e.Port = "0"
		}
	}

	e.finalized = true
}

func (e *RawEntry) splitPorts() (lp string, pp string, extra string) {
	portSplit := strings.Split(e.Port, ":")
	if len(portSplit) == 1 {
		pp = portSplit[0]
	} else {
		lp = portSplit[0]
		pp = portSplit[1]
		if len(portSplit) > 2 {
			extra = strings.Join(portSplit[2:], ":")
		}
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
