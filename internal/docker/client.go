package docker

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
)

type (
	SharedClient struct {
		*client.Client

		key      string
		refCount uint32
		closedOn int64
	}
)

var (
	clientMap   = make(map[string]*SharedClient, 5)
	clientMapMu sync.RWMutex

	clientOptEnvHost = []client.Opt{
		client.WithHostFromEnv(),
		client.WithAPIVersionNegotiation(),
	}
)

const (
	cleanInterval = 10 * time.Second
	clientTTLSecs = int64(10)
)

func init() {
	cleaner := task.RootTask("docker_clients_cleaner")
	go func() {
		ticker := time.NewTicker(cleanInterval)
		defer ticker.Stop()
		defer cleaner.Finish("program exit")

		for {
			select {
			case <-ticker.C:
				closeTimedOutClients()
			case <-cleaner.Context().Done():
				return
			}
		}
	}()

	task.OnProgramExit("docker_clients_cleanup", func() {
		clientMapMu.Lock()
		defer clientMapMu.Unlock()

		for _, c := range clientMap {
			delete(clientMap, c.key)
			c.Client.Close()
		}
	})
}

func closeTimedOutClients() {
	clientMapMu.Lock()
	defer clientMapMu.Unlock()

	now := time.Now().Unix()

	for _, c := range clientMap {
		if c.closedOn == 0 {
			continue
		}
		if c.refCount == 0 && now-c.closedOn > clientTTLSecs {
			delete(clientMap, c.key)
			c.Client.Close()
			logging.Debug().Str("host", c.key).Msg("docker client closed")
		}
	}
}

// if the client is still referenced, this is no-op.
func (c *SharedClient) Close() {
	atomic.StoreInt64(&c.closedOn, time.Now().Unix())
	atomic.AddUint32(&c.refCount, ^uint32(0))
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
func ConnectClient(host string) (*SharedClient, error) {
	clientMapMu.Lock()
	defer clientMapMu.Unlock()

	if client, ok := clientMap[host]; ok {
		atomic.StoreInt64(&client.closedOn, 0)
		atomic.AddUint32(&client.refCount, 1)
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
		refCount: 1,
	}

	defer logging.Debug().Str("host", host).Msg("docker client connected")

	clientMap[c.key] = c
	return c, nil
}
