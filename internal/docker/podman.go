package docker

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Resolve once per provider snapshot, including services with no idle timeout:
// the idlewatcher also reads Dependencies when it builds recursive watchers.
func resolvePodmanDependencies(ctx context.Context, shared *SharedClient, containers []*Container) {
	byID := make(map[string]*Container, len(containers))
	for _, c := range containers {
		byID[c.ContainerID] = c
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var group errgroup.Group
	group.SetLimit(8)
	for _, c := range containers {
		if c.ActualLabels["io.podman.compose.project"] == "" || len(Dependencies(c)) != 0 {
			continue
		}
		group.Go(func() error {
			ids, err := inspectPodmanDependencies(ctx, shared, c.ContainerID)
			if err != nil {
				log.Warn().Err(err).Str("container", c.ContainerName).Msg("failed to discover Podman dependencies")
				return nil
			}
			for _, id := range ids {
				dep := byID[id]
				if dep == nil {
					log.Warn().Str("container", c.ContainerName).Str("dependency", id).Msg("Podman dependency missing from container snapshot")
					continue
				}
				// Native dependencies also include shared namespace/infra containers.
				service := DockerComposeService(dep)
				if service != "" && DockerComposeProject(dep) == DockerComposeProject(c) && !slices.Contains(c.nativeDependencies, service) {
					c.nativeDependencies = append(c.nativeDependencies, service)
				}
			}
			if c.IdlewatcherConfig != nil {
				c.IdlewatcherConfig.DependsOn = Dependencies(c)
			}
			return nil
		})
	}
	_ = group.Wait()
}

func inspectPodmanDependencies(ctx context.Context, shared *SharedClient, id string) ([]string, error) {
	// Use the Docker client's actual endpoint, not SharedClient's display address:
	// agent connections use a fixed HTTPS hostname with their own TLS dialer.
	host, err := client.ParseHostURL(shared.Client.DaemonHost())
	if err != nil {
		return nil, err
	}
	httpClient := shared.HTTPClient()
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	base := "http://" + client.DummyHost
	if host.Scheme != "unix" && host.Scheme != "npipe" {
		scheme := "http"
		if host.Scheme == "https" || host.Scheme == "tls" || shared.cfg.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + host.Host + host.Path
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v4.0.0/libpod/containers/"+url.PathEscape(id)+"/json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Podman inspect: HTTP %d", resp.StatusCode)
	}
	var inspect struct {
		ID           string   `json:"Id"`
		Dependencies []string `json:"Dependencies"`
	}
	if err := json.UnmarshalRead(io.LimitReader(resp.Body, 8<<20), &inspect); err != nil {
		return nil, err
	}
	if inspect.ID != id || inspect.Dependencies == nil {
		return nil, fmt.Errorf("invalid Podman inspect response")
	}
	return inspect.Dependencies, nil
}
