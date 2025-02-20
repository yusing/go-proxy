package dockerapi

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/docker/docker/api/types/container"
	"github.com/yusing/go-proxy/internal/gperr"
	"github.com/yusing/go-proxy/internal/net/gphttp/gpwebsocket"
	"github.com/yusing/go-proxy/internal/net/gphttp/httpheaders"
)

type Container struct {
	Server string `json:"server"`
	Name   string `json:"name"`
	ID     string `json:"id"`
	Image  string `json:"image"`
	State  string `json:"state"`
}

func Containers(w http.ResponseWriter, r *http.Request) {
	if httpheaders.IsWebsocket(r.Header) {
		gpwebsocket.Periodic(w, r, 5*time.Second, func(conn *websocket.Conn) error {
			containers, err := listContainers(r.Context())
			if err != nil {
				return err
			}
			return wsjson.Write(r.Context(), conn, containers)
		})
	} else {
		containers, err := listContainers(r.Context())
		handleResult(w, err, containers)
	}
}

func listContainers(ctx context.Context) ([]Container, error) {
	ctx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()

	dockerClients, err := getDockerClients()
	if err != nil {
		return nil, err
	}
	defer closeAllClients(dockerClients)

	errs := gperr.NewBuilder("failed to get containers")
	containers := make([]Container, 0)
	for server, dockerClient := range dockerClients {
		conts, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true})
		if err != nil {
			errs.Add(err)
			continue
		}
		for _, cont := range conts {
			containers = append(containers, Container{
				Server: server,
				Name:   cont.Names[0],
				ID:     cont.ID,
				Image:  cont.Image,
				State:  cont.State,
			})
		}
	}
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})
	if err := errs.Error(); err != nil {
		gperr.LogError("failed to get containers", err)
		if len(containers) == 0 {
			return nil, err
		}
		return containers, nil
	}
	return containers, nil
}
