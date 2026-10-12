package docker

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/yusing/godoxy/agent/pkg/agent"
)

// labelPodmanComposeProject is set by podman-compose on every container it
// creates. It is the cheapest way to tell a podman-compose host from Docker
// without an extra API call.
const labelPodmanComposeProject = "io.podman.compose.project"

// libpodAPIVersion is the native Podman API version queried for dependencies.
// Podman 4 and newer serve it on the same socket as the Docker-compatible API.
const libpodAPIVersion = "v4.0.0"

const podmanRequestTimeout = 3 * time.Second

// podmanDependencies returns a podman-compose container's dependencies without
// requiring the proxy.depends_on label.
//
// podman-compose turns compose depends_on into podman --requires, which the
// Docker-compatible API does not expose, so it is read from the native libpod
// API and resolved back to the names the idle watcher matches on.
//
// It returns nil on Docker, on containers podman-compose did not create, on TLS
// providers, behind a Docker-API-only proxy (socket-proxy), and whenever the
// libpod API is unreachable.
func podmanDependencies(ctx context.Context, c *Container) []string {
	if c.ContainerID == "" || c.DockerCfg.TLS != nil {
		return nil
	}
	if cmp.Or(c.ActualLabels[labelPodmanComposeProject], c.Labels[labelPodmanComposeProject]) == "" {
		return nil
	}

	httpClient, base, ok := podmanEndpoint(ctx, c)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, podmanRequestTimeout)
	defer cancel()

	return fetchPodmanDependencies(ctx, httpClient, base, c.ContainerID)
}

// podmanEndpoint returns the root URL and HTTP client that reach the host's API,
// either through an agent or over the provider's own connection.
func podmanEndpoint(ctx context.Context, c *Container) (*http.Client, string, bool) {
	if agent.IsDockerHostAgent(c.DockerCfg.URL) {
		if c.Agent == nil {
			return nil, "", false
		}
		// The agent forwards any path to its DOCKER_SOCKET, which is Podman's
		// socket when Podman manages the host.
		return c.Agent.HTTPClient(), agent.DockerHost, true
	}

	cli, err := NewClient(ctx, c.DockerCfg)
	if err != nil {
		return nil, "", false
	}
	defer cli.Close()

	// cli.dial already reaches unix, tcp and connection helper hosts, so the URL
	// host below is never used to dial.
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return cli.dial(ctx) },
		},
	}, "http://podman", true
}

// fetchPodmanDependencies reads the --requires list of containerID, which Podman
// stores as container IDs, and maps them back to names.
func fetchPodmanDependencies(ctx context.Context, httpClient *http.Client, base, containerID string) []string {
	var inspect struct {
		Dependencies []string `json:"Dependencies"`
	}
	endpoint := base + "/" + libpodAPIVersion + "/libpod/containers/" + url.PathEscape(containerID) + "/json"
	if err := podmanGetJSON(ctx, httpClient, endpoint, &inspect); err != nil || len(inspect.Dependencies) == 0 {
		return nil
	}

	// The idle watcher resolves dependencies by name: the compose service name
	// where the provider reports one, the container name otherwise.
	var summaries []container.Summary
	if err := podmanGetJSON(ctx, httpClient, base+"/containers/json?all=true", &summaries); err != nil {
		return nil
	}

	names := make(map[string]string, len(summaries))
	for i, s := range summaries {
		if name := cmp.Or(
			s.Labels["com.docker.compose.service"],
			s.Labels["io.podman.compose.service"],
			containerHelper{&summaries[i]}.getName(),
		); name != "" {
			names[s.ID] = name
		}
	}

	deps := make([]string, 0, len(inspect.Dependencies))
	for _, id := range inspect.Dependencies {
		if name := names[id]; name != "" {
			deps = append(deps, name)
		}
	}
	return deps
}

func podmanGetJSON(ctx context.Context, httpClient *http.Client, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("podman: GET %s: %s", endpoint, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
