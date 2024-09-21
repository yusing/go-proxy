package docker

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
)

type Client struct {
	key      string
	refCount *atomic.Int32
	*client.Client

	l logrus.FieldLogger
}

func ParseDockerHostname(host string) (string, E.NestedError) {
	if host == common.DockerHostFromEnv {
		return host, nil
	} else if host == "" {
		return "localhost", nil
	}
	url, err := E.Check(client.ParseHostURL(host))
	if err != nil {
		return "", E.Invalid("host", host).With(err)
	}
	return url.Hostname(), nil
}

func (c Client) DaemonHostname() string {
	// DaemonHost should always return a valid host
	hostname, _ := ParseDockerHostname(c.DaemonHost())
	return hostname
}

func (c Client) Connected() bool {
	return c.Client != nil
}

// if the client is still referenced, this is no-op
func (c *Client) Close() error {
	if c.refCount.Add(-1) > 0 {
		return nil
	}

	clientMapMu.Lock()
	defer clientMapMu.Unlock()
	delete(clientMap, c.key)

	client := c.Client
	c.Client = nil

	c.l.Debugf("client closed")

	if client != nil {
		return client.Close()
	}
	return nil
}

// ConnectClient creates a new Docker client connection to the specified host.
//
// Returns existing client if available.
//
// Parameters:
//   - host: the host to connect to (either a URL or common.DockerHostFromEnv).
//
// Returns:
//   - Client: the Docker client connection.
//   - error: an error if the connection failed.
func ConnectClient(host string) (Client, E.NestedError) {
	clientMapMu.Lock()
	defer clientMapMu.Unlock()

	// check if client exists
	if client, ok := clientMap[host]; ok {
		client.refCount.Add(1)
		return client, nil
	}

	// create client
	var opt []client.Opt

	switch host {
	case common.DockerHostFromEnv:
		opt = clientOptEnvHost
	default:
		helper, err := E.Check(connhelper.GetConnectionHelper(host))
		if err.HasError() {
			return Client{}, E.UnexpectedError(err.Error())
		}
		if helper != nil {
			httpClient := &http.Client{
				Transport: &http.Transport{
					DialContext: helper.Dialer,
				},
			}
			opt = []client.Opt{
				client.WithHTTPClient(httpClient),
				client.WithHost(helper.Host),
				client.WithAPIVersionNegotiation(),
				client.WithDialContext(helper.Dialer),
			}
		} else {
			opt = []client.Opt{
				client.WithHost(host),
				client.WithAPIVersionNegotiation(),
			}
		}
	}

	client, err := E.Check(client.NewClientWithOpts(opt...))

	if err.HasError() {
		return Client{}, err
	}

	c := Client{
		Client:   client,
		key:      host,
		refCount: &atomic.Int32{},
		l:        logger.WithField("docker_client", client.DaemonHost()),
	}
	c.refCount.Add(1)
	c.l.Debugf("client connected")

	clientMap[host] = c
	return clientMap[host], nil
}

func CloseAllClients() {
	clientMapMu.Lock()
	defer clientMapMu.Unlock()
	for _, client := range clientMap {
		client.Close()
	}
	clientMap = make(map[string]Client)
	logger.Debug("closed all clients")
}

var (
	clientMap        map[string]Client = make(map[string]Client)
	clientMapMu      sync.Mutex
	clientOptEnvHost = []client.Opt{
		client.WithHostFromEnv(),
		client.WithAPIVersionNegotiation(),
	}

	logger = logrus.WithField("module", "docker")
)
