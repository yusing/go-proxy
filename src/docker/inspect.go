package docker

import (
	"context"
	"time"

	E "github.com/yusing/go-proxy/error"
)

func (c Client) Inspect(containerID string) (Container, E.NestedError) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	json, err := c.ContainerInspect(ctx, containerID)
	if err != nil {
		return Container{}, E.From(err)
	}
	return FromJson(json, c.key), nil
}
