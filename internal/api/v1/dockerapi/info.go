package dockerapi

import (
	"context"
	"encoding/json"
	"net/http"

	dockerSystem "github.com/docker/docker/api/types/system"
	"github.com/yusing/go-proxy/internal/gperr"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type DockerInfo dockerSystem.Info

func (d *DockerInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"host": d.Name,
		"containers": map[string]int{
			"total":   d.Containers,
			"running": d.ContainersRunning,
			"paused":  d.ContainersPaused,
			"stopped": d.ContainersStopped,
		},
		"images":  d.Images,
		"n_cpu":   d.NCPU,
		"memory":  strutils.FormatByteSizeWithUnit(d.MemTotal),
		"version": d.ServerVersion,
	})
}

func Info(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), reqTimeout)
	defer cancel()

	dockerClients, ok := getDockerClientsWithErrHandling(w)
	if !ok {
		return
	}
	defer closeAllClients(dockerClients)

	errs := gperr.NewBuilder("failed to get docker info")

	dockerInfos := make([]DockerInfo, len(dockerClients))
	i := 0
	for name, dockerClient := range dockerClients {
		info, err := dockerClient.Info(ctx)
		if err != nil {
			errs.Add(err)
			continue
		}
		info.Name = name
		dockerInfos[i] = DockerInfo(info)
		i++
	}

	handleResult(w, errs.Error(), dockerInfos)
}
