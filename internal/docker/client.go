package docker

import (
	"errors"
	"net/http"
	"sync"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	Client       = *SharedClient
	SharedClient struct {
		*client.Client

		key      string
		refCount *U.RefCount

		l zerolog.Logger
	}
)

var (
	clientMap   F.Map[string, Client] = F.NewMapOf[string, Client]()
	clientMapMu sync.Mutex

	clientOptEnvHost = []client.Opt{
		client.WithHostFromEnv(),
		client.WithAPIVersionNegotiation(),
	}
)

func init() {
	task.GlobalTask("close docker clients").OnFinished("", func() {
		clientMap.RangeAllParallel(func(_ string, c Client) {
			if c.Connected() {
				c.Client.Close()
			}
		})
	})
}

func (c *SharedClient) Connected() bool {
	return c != nil && c.Client != nil
}

// if the client is still referenced, this is no-op.
func (c *SharedClient) Close() {
	if c.Connected() {
		c.refCount.Sub()
	}
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
func ConnectClient(host string) (Client, error) {
	clientMapMu.Lock()
	defer clientMapMu.Unlock()

	// check if client exists
	if client, ok := clientMap.Load(host); ok {
		client.refCount.Add()
		return client, nil
	}

	// create client
	var opt []client.Opt

	switch host {
	case "":
		return nil, errors.New("empty docker host")
	case common.DockerHostFromEnv:
		opt = clientOptEnvHost
	default:
		helper, err := connhelper.GetConnectionHelper(host)
		if err != nil {
			logging.Panic().Err(err).Msg("failed to get connection helper")
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

	client, err := client.NewClientWithOpts(opt...)
	if err != nil {
		return nil, err
	}

	c := &SharedClient{
		Client:   client,
		key:      host,
		refCount: U.NewRefCounter(),
		l:        logger.With().Str("address", client.DaemonHost()).Logger(),
	}
	c.l.Trace().Msg("client connected")

	clientMap.Store(host, c)

	go func() {
		<-c.refCount.Zero()
		clientMap.Delete(c.key)

		if c.Connected() {
			c.Client.Close()
			c.l.Trace().Msg("client closed")
		}
	}()
	return c, nil
}
