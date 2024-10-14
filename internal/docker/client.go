package docker

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type Client struct {
	*client.Client

	key      string
	refCount *atomic.Int32

	l logrus.FieldLogger
}

var (
	clientMap   F.Map[string, Client] = F.NewMapOf[string, Client]()
	clientMapMu sync.Mutex

	clientOptEnvHost = []client.Opt{
		client.WithHostFromEnv(),
		client.WithAPIVersionNegotiation(),
	}
)

func (c Client) Connected() bool {
	return c.Client != nil
}

// if the client is still referenced, this is no-op.
func (c *Client) Close() error {
	if c.refCount.Add(-1) > 0 {
		return nil
	}

	clientMap.Delete(c.key)

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
	if client, ok := clientMap.Load(host); ok {
		client.refCount.Add(1)
		return client, nil
	}

	// create client
	var opt []client.Opt

	switch host {
	case "":
		return Client{}, E.Invalid("docker host", "empty")
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

	clientMap.Store(host, c)
	return c, nil
}

func CloseAllClients() {
	clientMap.RangeAllParallel(func(_ string, c Client) {
		c.Client.Close()
	})
	clientMap.Clear()
	logger.Debug("closed all clients")
}
