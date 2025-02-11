package agent

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/agent/pkg/certs"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/pkg"
	"golang.org/x/net/context"
)

type (
	AgentConfig struct {
		Addr string

		httpClient *http.Client
		tlsConfig  *tls.Config
		name       string
		l          zerolog.Logger
	}
)

const (
	EndpointVersion    = "/version"
	EndpointName       = "/name"
	EndpointProxyHTTP  = "/proxy/http"
	EndpointHealth     = "/health"
	EndpointLogs       = "/logs"
	EndpointSystemInfo = "/system-info"

	AgentHost = certs.CertsDNSName

	APIEndpointBase = "/godoxy/agent"
	APIBaseURL      = "https://" + AgentHost + APIEndpointBase

	DockerHost = "https://" + AgentHost

	FakeDockerHostPrefix    = "agent://"
	FakeDockerHostPrefixLen = len(FakeDockerHostPrefix)
)

var (
	agents     = functional.NewMapOf[string, *AgentConfig]()
	agentMapMu sync.RWMutex
)

var (
	HTTPProxyURL         = types.MustParseURL(APIBaseURL + EndpointProxyHTTP)
	HTTPProxyURLStripLen = len(APIEndpointBase + EndpointProxyHTTP)
)

func IsDockerHostAgent(dockerHost string) bool {
	return strings.HasPrefix(dockerHost, FakeDockerHostPrefix)
}

func GetAgentFromDockerHost(dockerHost string) (*AgentConfig, bool) {
	if !IsDockerHostAgent(dockerHost) {
		return nil, false
	}
	return agents.Load(dockerHost[FakeDockerHostPrefixLen:])
}

func (cfg *AgentConfig) FakeDockerHost() string {
	return FakeDockerHostPrefix + cfg.Name()
}

func (cfg *AgentConfig) Parse(addr string) error {
	cfg.Addr = addr
	return cfg.load()
}

func (cfg *AgentConfig) errIfNameExists() E.Error {
	agentMapMu.RLock()
	defer agentMapMu.RUnlock()
	agent, ok := agents.Load(cfg.Name())
	if ok {
		return E.Errorf("agent with name %s (%s) already exists", cfg.Name(), agent.Addr)
	}
	return nil
}

func withoutBuildTime(version string) string {
	return strings.Split(version, "-")[0]
}

func checkVersion(a, b string) bool {
	return withoutBuildTime(a) == withoutBuildTime(b)
}

func (cfg *AgentConfig) Remove() {
	agentMapMu.Lock()
	defer agentMapMu.Unlock()
	agents.Delete(cfg.Name())
}

func (cfg *AgentConfig) load() E.Error {
	certData, err := os.ReadFile(certs.AgentCertsFilename(cfg.Addr))
	if err != nil {
		if os.IsNotExist(err) {
			return E.Errorf("agents certs not found, did you run `godoxy new-agent %s ...`?", cfg.Addr)
		}
		return E.Wrap(err)
	}

	ca, crt, key, err := certs.ExtractCert(certData)
	if err != nil {
		return E.Wrap(err)
	}

	clientCert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		return E.Wrap(err)
	}

	// create tls config
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(ca)
	if !ok {
		return E.New("invalid CA certificate")
	}

	cfg.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
	}

	// create transport and http client
	cfg.httpClient = cfg.NewHTTPClient()

	ctx, cancel := context.WithTimeout(task.RootContext(), 5*time.Second)
	defer cancel()

	// check agent version
	version, _, err := cfg.Fetch(ctx, EndpointVersion)
	if err != nil {
		return E.Wrap(err)
	}

	if !checkVersion(string(version), pkg.GetVersion()) {
		return E.Errorf("agent version mismatch: server: %s, agent: %s", pkg.GetVersion(), string(version))
	}

	// get agent name
	name, _, err := cfg.Fetch(ctx, EndpointName)
	if err != nil {
		return E.Wrap(err)
	}

	// check if agent name is already used
	cfg.name = string(name)
	if err := cfg.errIfNameExists(); err != nil {
		return err
	}

	cfg.l = logging.With().Str("agent", cfg.name).Logger()

	agents.Store(cfg.name, cfg)
	return nil
}

func (cfg *AgentConfig) NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: cfg.Transport(),
	}
}

func (cfg *AgentConfig) Transport() *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if addr != AgentHost+":443" {
				return nil, &net.AddrError{Err: "invalid address", Addr: addr}
			}
			return gphttp.DefaultDialer.DialContext(ctx, network, cfg.Addr)
		},
		TLSClientConfig: cfg.tlsConfig,
	}
}

func (cfg *AgentConfig) Name() string {
	return cfg.name
}

func (cfg *AgentConfig) String() string {
	return "agent@" + cfg.Name()
}

func (cfg *AgentConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"name": cfg.Name(),
		"addr": cfg.Addr,
	})
}
