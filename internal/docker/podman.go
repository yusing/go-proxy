package docker

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

// labelPodmanComposeProject is set by podman-compose on every container it
// creates, so its presence identifies a podman-compose host without an API call.
const labelPodmanComposeProject = "io.podman.compose.project"

// libpodAPIVersion is the native Podman API version queried for dependencies.
// Podman accepts any version segment, and v4.0.0 is the oldest libpod API, so
// Podman 4 and newer answer it.
const libpodAPIVersion = "v4.0.0"

const podmanRequestTimeout = 3 * time.Second
const podmanResponseLimit = 8 << 20

type podmanNamesKey struct{}

// WithPodmanContainers stores container names from the provider's snapshot in
// ctx, so dependency resolution can map IDs to names without listing again.
func WithPodmanContainers(ctx context.Context, summaries []container.Summary) context.Context {
	return context.WithValue(ctx, podmanNamesKey{}, podmanContainerNames(summaries))
}

func podmanContainerNames(summaries []container.Summary) map[string]string {
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
	return names
}

// podmanDependencies returns a podman-compose container's dependencies.
//
// podman-compose stores compose depends_on as podman --requires, which the
// Docker-compatible API does not report. This reads the list from the native
// libpod API and resolves the container IDs to the names the idle watcher
// matches on.
//
// A failure is logged and treated as no dependencies, and the container's routes
// are still built.
func podmanDependencies(ctx context.Context, c *Container) []string {
	if c.ContainerID == "" || cmp.Or(c.ActualLabels[labelPodmanComposeProject], c.Labels[labelPodmanComposeProject]) == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, podmanRequestTimeout)
	defer cancel()

	httpClient, base, cleanup, err := podmanEndpoint(ctx, c)
	if err != nil {
		log.Warn().Err(err).Str("container", c.ContainerName).Msg("podman dependencies")
		return nil
	}
	defer cleanup()
	deps, err := fetchPodmanDependencies(ctx, httpClient, base, c.ContainerID)
	if err != nil {
		log.Warn().Err(err).Str("container", c.ContainerName).Msg("podman dependencies")
	}
	return deps
}

// podmanEndpoint returns an HTTP client and base URL that reach the host's API,
// through an agent or the provider's connection.
func podmanEndpoint(ctx context.Context, c *Container) (*http.Client, string, func(), error) {
	shared, err := NewClient(ctx, c.DockerCfg)
	if err != nil {
		return nil, "", nil, err
	}
	host, err := client.ParseHostURL(shared.Client.DaemonHost())
	if err != nil {
		shared.Close()
		return nil, "", nil, err
	}
	httpClient := shared.HTTPClient()
	base := "http://" + client.DummyHost // NOSONAR: unix/npipe/ssh transports stay on the host
	if host.Scheme != "unix" && host.Scheme != "npipe" && host.Scheme != "ssh" {
		scheme := "http"
		if transport, ok := httpClient.Transport.(*http.Transport); host.Scheme == "https" || host.Scheme == "tls" || ok && transport.TLSClientConfig != nil {
			scheme = "https"
		}
		base = scheme + "://" + host.Host + host.Path
	}
	return httpClient, base, shared.Close, nil
}

func validPodmanID(id string) bool {
	if len(id) != 64 {
		return false
	}
	for _, ch := range id {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
			return false
		}
	}
	return true
}

// fetchPodmanDependencies reads the --requires list for containerID, which
// Podman reports as full container IDs, and returns the matching compose
// service names.
func fetchPodmanDependencies(ctx context.Context, httpClient *http.Client, base, containerID string) ([]string, error) {
	if !validPodmanID(containerID) {
		return nil, fmt.Errorf("invalid container ID")
	}
	var inspect struct {
		ID           string `json:"Id"`
		Dependencies []string
	}
	endpoint := base + "/" + libpodAPIVersion + "/libpod/containers/" + url.PathEscape(containerID) + "/json"
	if err := podmanGetJSON(ctx, httpClient, endpoint, &inspect); err != nil {
		return nil, err
	}
	if inspect.ID != containerID || inspect.Dependencies == nil {
		return nil, fmt.Errorf("invalid libpod inspect response")
	}
	if len(inspect.Dependencies) == 0 {
		return nil, nil
	}

	names, ok := ctx.Value(podmanNamesKey{}).(map[string]string)
	if !ok {
		var summaries []container.Summary
		if err := podmanGetJSON(ctx, httpClient, base+"/containers/json?all=true", &summaries); err != nil {
			return nil, err
		}
		names = podmanContainerNames(summaries)
	}
	deps := make([]string, 0, len(inspect.Dependencies))
	seen := make(map[string]bool, len(inspect.Dependencies))
	for _, id := range inspect.Dependencies {
		if !validPodmanID(id) || id == containerID {
			return nil, fmt.Errorf("invalid dependency ID %q", id)
		}
		name := names[id]
		if name == "" {
			// The container may have started after the snapshot was taken.
			log.Warn().Str("container", containerID).Str("dependency", id).Msg("podman dependency not in snapshot, skipping")
			continue
		}
		if !seen[name] {
			deps = append(deps, name)
			seen[name] = true
		}
	}
	return deps, nil
}

func podmanGetJSON(ctx context.Context, httpClient *http.Client, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	// Redirects are disabled even when the caller supplied the client.
	client := *httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("podman: unexpected HTTP status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, podmanResponseLimit+1))
	if err != nil {
		return err
	}
	if len(body) > podmanResponseLimit {
		return fmt.Errorf("podman: response exceeds %d bytes", podmanResponseLimit)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("podman: invalid JSON: %w", err)
	}
	if bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		return fmt.Errorf("podman: null response")
	}
	return nil
}
