package docker

import (
	"net/http"
	"sync"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
)

type Client = *client.Client

// ConnectClient creates a new Docker client connection to the specified host.
//
// Returns existing client if available.
//
// Parameters:
//   - host: the host to connect to (either a URL or "FROM_ENV").
//
// Returns:
//   - Client: the Docker client connection.
//   - error: an error if the connection failed.
func ConnectClient(host string) (Client, E.NestedError) {
	clientMapMu.Lock()
	defer clientMapMu.Unlock()

	// check if client exists
	if client, ok := clientMap[host]; ok {
		return client, E.Nil()
	}

	// create client
	var opt []client.Opt

	switch host {
	case common.DockerHostFromEnv:
		opt = clientOptEnvHost
	default:
		helper, err := E.Check(connhelper.GetConnectionHelper(host))
		if err.IsNotNil() {
			logger.Fatalf("unexpected error: %s", err)
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

	if err.IsNotNil() {
		return nil, err
	}

	clientMap[host] = client
	return client, E.Nil()
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

var clientMap map[string]Client = make(map[string]Client)
var clientMapMu sync.Mutex

var clientOptEnvHost = []client.Opt{
	client.WithHostFromEnv(),
	client.WithAPIVersionNegotiation(),
}

var logger = logrus.WithField("module", "docker")
