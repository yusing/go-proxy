package docker

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/yusing/godoxy/internal/types"
)

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

func ListContainers(ctx context.Context, dockerCfg types.DockerProviderConfig) ([]*Container, error) {
	dockerClient, err := NewClient(ctx, dockerCfg)
	if err != nil {
		return nil, err
	}
	defer dockerClient.Close()

	summaries, err := dockerClient.ContainerList(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	containers := make([]*Container, len(summaries))
	for i := range summaries {
		containers[i] = FromDocker(ctx, &summaries[i], dockerCfg)
	}
	resolvePodmanDependencies(ctx, dockerClient, containers)
	return containers, nil
}

func IsErrConnectionFailed(err error) bool {
	return client.IsErrConnectionFailed(err)
}
