package docker

import (
	"context"
	"errors"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
)

func Inspect(dockerHost string, containerID string) (*Container, E.NestedError) {
	client, err := ConnectClient(dockerHost)
	defer client.Close()

	if err.HasError() {
		return nil, E.FailWith("connect to docker", err)
	}

	return client.Inspect(containerID)
}

func (c Client) Inspect(containerID string) (*Container, E.NestedError) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 3*time.Second, errors.New("docker container inspect timeout"))
	defer cancel()

	json, err := c.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, E.From(err)
	}
	return FromJSON(json, c.key), nil
}
