package docker

import (
	"context"
	"errors"
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
	// created|restarting|running|removing|paused|exited|dead
	// Filters: filters.NewArgs(
	// 	filters.Arg("status", "created"),
	// 	filters.Arg("status", "restarting"),
	// 	filters.Arg("status", "running"),
	// 	filters.Arg("status", "paused"),
	// 	filters.Arg("status", "exited"),
	// ),
	All: true,
}

func GetClientInfo(clientHost string, getContainer bool) (*ClientInfo, E.NestedError) {
	dockerClient, err := ConnectClient(clientHost)
	if err.HasError() {
		return nil, E.FailWith("connect to docker", err)
	}
	defer dockerClient.Close()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 3*time.Second, errors.New("docker client connection timeout"))
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
