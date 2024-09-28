package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	E "github.com/yusing/go-proxy/internal/error"
)

type ClientInfo struct {
	Client     Client
	Containers []types.Container
}

var listOptions = container.ListOptions{
	// Filters: filters.NewArgs(
	// 	filters.Arg("health", "healthy"),
	// 	filters.Arg("health", "none"),
	// 	filters.Arg("health", "starting"),
	// ),
	All: true,
}

func GetClientInfo(clientHost string, getContainer bool) (*ClientInfo, E.NestedError) {
	dockerClient, err := ConnectClient(clientHost)
	if err.HasError() {
		return nil, E.FailWith("connect to docker", err)
	}
	defer dockerClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var containers []types.Container
	if getContainer {
		containers, err = E.Check(dockerClient.ContainerList(ctx, listOptions))
		if err.HasError() {
			return nil, E.FailWith("list containers", err)
		}
	}

	return &ClientInfo{
		Client:     dockerClient,
		Containers: containers,
	}, nil
}

func IsErrConnectionFailed(err error) bool {
	return client.IsErrConnectionFailed(err)
}
