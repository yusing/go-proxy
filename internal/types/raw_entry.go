package types

import (
	"strconv"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	D "github.com/yusing/go-proxy/internal/docker"
	H "github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	RawEntry struct {
		// raw entry object before validation
		// loaded from docker labels or yaml file
		Alias        string              `json:"-" yaml:"-"`
		Scheme       string              `json:"scheme" yaml:"scheme"`
		Host         string              `json:"host" yaml:"host"`
		Port         string              `json:"port" yaml:"port"`
		NoTLSVerify  bool                `json:"no_tls_verify,omitempty" yaml:"no_tls_verify"` // https proxy only
		PathPatterns []string            `json:"path_patterns,omitempty" yaml:"path_patterns"` // http(s) proxy only
		LoadBalance  loadbalancer.Config `json:"load_balance" yaml:"load_balance"`
		Middlewares  D.NestedLabelMap    `json:"middlewares,omitempty" yaml:"middlewares"`
		Homepage     *H.HomePageItem     `json:"homepage,omitempty" yaml:"homepage"`

		/* Docker only */
		*D.ProxyProperties `json:"proxy_properties" yaml:"-"`
	}

	RawEntries = F.Map[string, *RawEntry]
)

var NewProxyEntries = F.NewMapOf[string, *RawEntry]

func (e *RawEntry) FillMissingFields() {
	isDocker := e.ProxyProperties != nil
	if !isDocker {
		e.ProxyProperties = &D.ProxyProperties{}
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
		if p, ok := F.FirstValueOf(e.PrivatePortMapping); ok {
			pp = U.PortString(p.PrivatePort)
		} else if !isDocker {
			pp = "80"
		}
	}

	// replace private port with public port (if any)
	if isDocker && e.NetworkMode != "host" {
		if p, ok := e.PrivatePortMapping[pp]; ok {
			pp = U.PortString(p.PublicPort)
		}
		if _, ok := e.PublicPortMapping[pp]; !ok { // port is not exposed, but specified
			// try to fallback to first public port
			if p, ok := F.FirstValueOf(e.PublicPortMapping); ok {
				pp = U.PortString(p.PublicPort)
			}
		}
	}

	if e.Scheme == "" && isDocker {
		if p, ok := e.PublicPortMapping[pp]; ok && p.Type == "udp" {
			e.Scheme = "udp"
		}
	}

	if e.Scheme == "" {
		switch {
		case lp != "":
			e.Scheme = "tcp"
		case strings.HasSuffix(pp, "443"):
			e.Scheme = "https"
		default:
			// assume its http
			e.Scheme = "http"
		}
	}

	if e.Host == "" {
		e.Host = "localhost"
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

	if e.Port == "" {
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
