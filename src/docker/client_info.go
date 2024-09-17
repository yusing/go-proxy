package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	E "github.com/yusing/go-proxy/error"
)

type ClientInfo struct {
	Host       string
	Containers []types.Container
}

func GetClientInfo(clientHost string) (*ClientInfo, E.NestedError) {
	dockerClient, err := ConnectClient(clientHost)
	if err.HasError() {
		return nil, E.Failure("create docker client").With(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	containers, err := E.Check(dockerClient.ContainerList(ctx, container.ListOptions{}))
	if err.HasError() {
		return nil, E.Failure("list containers").With(err)
	}

	// extract host from docker client url
	// since the services being proxied to
	// should have the same IP as the docker client
	url, err := E.Check(client.ParseHostURL(dockerClient.DaemonHost()))
	if err.HasError() {
		return nil, E.Invalid("host url", dockerClient.DaemonHost()).With(err)
	}
	if url.Scheme == "unix" {
		return &ClientInfo{Host: "localhost", Containers: containers}, E.Nil()
	}
	return &ClientInfo{Host: url.Hostname(), Containers: containers}, E.Nil()
}

func IsErrConnectionFailed(err error) bool {
	return client.IsErrConnectionFailed(err)
}
