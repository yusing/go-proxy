package route

import (
	"fmt"
	"strings"

	"github.com/yusing/go-proxy/agent/pkg/agent"
	"github.com/yusing/go-proxy/internal/docker"
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	"github.com/yusing/go-proxy/internal/homepage"
	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/watcher/health"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/accesslog"
	loadbalance "github.com/yusing/go-proxy/internal/net/http/loadbalancer/types"
	"github.com/yusing/go-proxy/internal/route/rules"
	"github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/utils"
)

type (
	Route struct {
		_ utils.NoCopy

		Alias  string       `json:"alias"`
		Scheme types.Scheme `json:"scheme,omitempty"`
		Host   string       `json:"host,omitempty"`
		Port   types.Port   `json:"port,omitempty"`
		Root   string       `json:"root,omitempty"`

		types.HTTPConfig
		PathPatterns []string                   `json:"path_patterns,omitempty"`
		Rules        rules.Rules                `json:"rules,omitempty" validate:"omitempty,unique=Name"`
		HealthCheck  *health.HealthCheckConfig  `json:"healthcheck,omitempty"`
		LoadBalance  *loadbalance.Config        `json:"load_balance,omitempty"`
		Middlewares  map[string]docker.LabelMap `json:"middlewares,omitempty"`
		Homepage     *homepage.Item             `json:"homepage,omitempty"`
		AccessLog    *accesslog.Config          `json:"access_log,omitempty"`

		Metadata `deserialize:"-"`
	}

	Metadata struct {
		/* Docker only */
		Container *docker.Container `json:"container,omitempty"`
		Provider  string            `json:"provider,omitempty"`

		// private fields
		LisURL      *net.URL            `json:"lurl,omitempty"`
		ProxyURL    *net.URL            `json:"purl,omitempty"`
		Idlewatcher *idlewatcher.Config `json:"idlewatcher,omitempty"`

		impl        types.Route
		isValidated bool
	}
	Routes map[string]*Route
)

func (r Routes) Contains(alias string) bool {
	_, ok := r[alias]
	return ok
}

func (r *Route) Validate() (err E.Error) {
	if r.isValidated {
		return nil
	}
	r.isValidated = true
	r.Finalize()

	errs := E.NewBuilder("entry validation failed")

	switch r.Scheme {
	case types.SchemeFileServer:
		r.impl, err = NewFileServer(r)
		if err != nil {
			errs.Add(err)
		}
	case types.SchemeHTTP, types.SchemeHTTPS:
		if r.Port.Listening != 0 {
			errs.Addf("unexpected listening port for %s scheme", r.Scheme)
		}
		fallthrough
	case types.SchemeTCP, types.SchemeUDP:
		r.LisURL = E.Collect(errs, net.ParseURL, fmt.Sprintf("%s://%s:%d", r.Scheme, r.Host, r.Port.Listening))
		fallthrough
	default:
		if r.LoadBalance != nil && r.LoadBalance.Link == "" {
			r.LoadBalance = nil
		}
		r.ProxyURL = E.Collect(errs, net.ParseURL, fmt.Sprintf("%s://%s:%d", r.Scheme, r.Host, r.Port.Proxy))
		r.Idlewatcher = E.Collect(errs, idlewatcher.ValidateConfig, r.Container)
	}

	if !r.UseHealthCheck() && (r.UseLoadBalance() || r.UseIdleWatcher()) {
		errs.Adds("healthCheck.disable cannot be true when loadbalancer or idlewatcher is enabled")
	}

	if errs.HasError() {
		return errs.Error()
	}

	switch r.Scheme {
	case types.SchemeFileServer:
		r.impl, err = NewFileServer(r)
	case types.SchemeHTTP, types.SchemeHTTPS:
		r.impl, err = NewReverseProxyRoute(r)
	case types.SchemeTCP, types.SchemeUDP:
		r.impl, err = NewStreamRoute(r)
	default:
		panic(fmt.Errorf("unexpected scheme %s for alias %s", r.Scheme, r.Alias))
	}

	return err
}

func (r *Route) Start(parent task.Parent) (err E.Error) {
	if r.impl == nil {
		return E.New("route not initialized")
	}
	return r.impl.Start(parent)
}

func (r *Route) Finish(reason any) {
	if r.impl == nil {
		return
	}
	r.impl.Finish(reason)
	r.impl = nil
}

func (r *Route) Started() bool {
	return r.impl != nil
}

func (r *Route) ProviderName() string {
	return r.Provider
}

func (r *Route) TargetName() string {
	return r.Alias
}

func (r *Route) TargetURL() *net.URL {
	return r.ProxyURL
}

func (r *Route) Type() types.RouteType {
	switch r.Scheme {
	case types.SchemeHTTP, types.SchemeHTTPS, types.SchemeFileServer:
		return types.RouteTypeHTTP
	case types.SchemeTCP, types.SchemeUDP:
		return types.RouteTypeStream
	}
	panic(fmt.Errorf("unexpected scheme %s for alias %s", r.Scheme, r.Alias))
}

func (r *Route) Agent() *agent.AgentConfig {
	if r.Container == nil {
		return nil
	}
	return r.Container.Agent
}

func (r *Route) IsAgent() bool {
	return r.Container != nil && r.Container.Agent != nil
}

func (r *Route) HealthMonitor() health.HealthMonitor {
	return r.impl.HealthMonitor()
}

func (r *Route) IdlewatcherConfig() *idlewatcher.Config {
	return r.Idlewatcher
}

func (r *Route) HealthCheckConfig() *health.HealthCheckConfig {
	return r.HealthCheck
}

func (r *Route) LoadBalanceConfig() *loadbalance.Config {
	return r.LoadBalance
}

func (r *Route) HomepageConfig() *homepage.Item {
	return r.Homepage
}

func (r *Route) ContainerInfo() *docker.Container {
	return r.Container
}

func (r *Route) IsDocker() bool {
	if r.Container == nil {
		return false
	}
	return r.Container.ContainerID != ""
}

func (r *Route) IsZeroPort() bool {
	return r.Port.Proxy == 0
}

func (r *Route) ShouldExclude() bool {
	if r.Container != nil {
		switch {
		case r.Container.IsExcluded:
			return true
		case r.IsZeroPort() && !r.UseIdleWatcher():
			return true
		case r.Container.IsDatabase && !r.Container.IsExplicit:
			return true
		case strings.HasPrefix(r.Container.ContainerName, "buildx_"):
			return true
		}
	} else if r.IsZeroPort() {
		return true
	}
	if strings.HasPrefix(r.Alias, "x-") ||
		strings.HasSuffix(r.Alias, "-old") {
		return true
	}
	return false
}

func (r *Route) UseLoadBalance() bool {
	return r.LoadBalance != nil && r.LoadBalance.Link != ""
}

func (r *Route) UseIdleWatcher() bool {
	return r.Idlewatcher != nil && r.Idlewatcher.IdleTimeout > 0
}

func (r *Route) UseHealthCheck() bool {
	return !r.HealthCheck.Disable
}

func (r *Route) UseAccessLog() bool {
	return r.AccessLog != nil
}

func (r *Route) Finalize() {
	isDocker := r.Container != nil
	cont := r.Container

	if r.Host == "" {
		switch {
		case !isDocker:
			r.Host = "localhost"
		case cont.PrivateHostname != "":
			r.Host = cont.PrivateHostname
		case cont.PublicHostname != "":
			r.Host = cont.PublicHostname
		}
	}

	lp, pp := r.Port.Listening, r.Port.Proxy

	if isDocker {
		scheme, port, ok := getSchemePortByImageName(cont.ImageName, pp)
		if ok {
			if r.Scheme == "" {
				r.Scheme = types.Scheme(scheme)
			}
			if pp == 0 {
				pp = port
			}
		}
	}

	if scheme, port, ok := getSchemePortByAlias(r.Alias, pp); ok {
		if r.Scheme == "" {
			r.Scheme = types.Scheme(scheme)
		}
		if pp == 0 {
			pp = port
		}
	}

	if pp == 0 {
		switch {
		case isDocker:
			pp = lowestPort(cont.PrivatePortMapping)
			if pp == 0 {
				pp = lowestPort(cont.PublicPortMapping)
			}
		case r.Scheme == "https":
			pp = 443
		default:
			pp = 80
		}
	}

	if isDocker {
		if r.Scheme == "" {
			for _, p := range cont.PublicPortMapping {
				if p.PrivatePort == uint16(pp) && p.Type == "udp" {
					r.Scheme = "udp"
					break
				}
			}
		}
		// replace private port with public port if using public IP.
		if r.Host == cont.PublicHostname {
			if p, ok := cont.PrivatePortMapping[pp]; ok {
				pp = int(p.PublicPort)
			}
		} else {
			// replace public port with private port if using private IP.
			if p, ok := cont.PublicPortMapping[pp]; ok {
				pp = int(p.PrivatePort)
			}
		}
	}

	if r.Scheme == "" {
		switch {
		case lp != 0:
			r.Scheme = "tcp"
		case pp%1000 == 443:
			r.Scheme = "https"
		default: // assume its http
			r.Scheme = "http"
		}
	}

	r.Port.Listening, r.Port.Proxy = lp, pp

	if r.HealthCheck == nil {
		r.HealthCheck = health.DefaultHealthConfig
	}

	if !r.HealthCheck.Disable {
		if r.HealthCheck.Interval == 0 {
			r.HealthCheck.Interval = common.HealthCheckIntervalDefault
		}
		if r.HealthCheck.Timeout == 0 {
			r.HealthCheck.Timeout = common.HealthCheckTimeoutDefault
		}
	}

	if isDocker && cont.IdleTimeout != "" {
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

	if r.Homepage.IsEmpty() {
		r.Homepage = homepage.NewItem(r.Alias)
	}
}

func lowestPort(ports map[int]dockertypes.Port) (res int) {
	cmp := (uint16)(65535)
	for port, v := range ports {
		if v.PrivatePort < cmp {
			cmp = v.PrivatePort
			res = port
		}
	}
	return
}
