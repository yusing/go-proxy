package docker

import (
	"net/http"
	"sync"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	Client       = *SharedClient
	SharedClient struct {
		*client.Client

		key      string
		refCount *U.RefCount

		l logrus.FieldLogger
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
	go func() {
		task := common.NewTask("close all docker client")
		defer task.Finished()
		for {
			select {
			case <-task.Context().Done():
				clientMap.RangeAllParallel(func(_ string, c Client) {
					c.Client.Close()
				})
				clientMap.Clear()
				return
			}
		}
	}()
}

func (c *SharedClient) Connected() bool {
	return c != nil && c.Client != nil
}

// if the client is still referenced, this is no-op.
func (c *SharedClient) Close() error {
	if !c.Connected() {
		return nil
	}

	c.refCount.Sub()
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
		client.refCount.Add()
		return client, nil
	}

	// create client
	var opt []client.Opt

	switch host {
	case "":
		return nil, E.Invalid("docker host", "empty")
	case common.DockerHostFromEnv:
		opt = clientOptEnvHost
	default:
		helper, err := E.Check(connhelper.GetConnectionHelper(host))
		if err.HasError() {
			return nil, E.UnexpectedError(err.Error())
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
		return nil, err
	}

	c := &SharedClient{
		Client:   client,
		key:      host,
		refCount: U.NewRefCounter(),
		l:        logger.WithField("docker_client", client.DaemonHost()),
	}
	c.l.Debugf("client connected")

	clientMap.Store(host, c)

	go func() {
		<-c.refCount.Zero()
		clientMap.Delete(c.key)

		if c.Client != nil {
			c.Client.Close()
			c.Client = nil
			c.l.Debugf("client closed")
		}
	}()
	return c, nil
}

func CloseAllClients() {
	clientMap.RangeAllParallel(func(_ string, c Client) {
		c.Client.Close()
	})
	clientMap.Clear()
	logger.Debug("closed all clients")
}
