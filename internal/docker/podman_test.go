package docker

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/internal/types"
)

func TestListContainersPodmanDependencies(t *testing.T) {
	summaries := []container.Summary{}
	for i, service := range []string{"app", "worker", "db", "explicit", "compose"} {
		labels := map[string]string{
			"com.docker.compose.project": "stack",
			"com.docker.compose.service": service,
			"io.podman.compose.project":  "stack",
		}
		if service != "worker" && service != "db" {
			labels[LabelIdleTimeout] = "1m"
		}
		if service == "explicit" {
			labels[LabelDependsOn] = "manual"
			labels["com.docker.compose.depends_on"] = "ignored"
		}
		if service == "compose" {
			labels["com.docker.compose.depends_on"] = "db:service_healthy"
			delete(labels, "io.podman.compose.project")
		}
		summaries = append(summaries, container.Summary{ID: strings.Repeat(fmt.Sprint(i+1), 64), Names: []string{"/stack_" + service + "_1"}, State: "exited", Labels: labels})
	}
	summaries = append(summaries, container.Summary{ID: "infra", Names: []string{"/pod_infra"}, State: "exited"})
	var inspections atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("API-Version", "1.47")
			return
		}
		if strings.HasSuffix(r.URL.Path, "/containers/json") {
			require.NoError(t, json.MarshalWrite(w, &summaries))
			return
		}
		require.Equal(t, http.MethodGet, r.Method)
		inspections.Add(1)
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v4.0.0/libpod/containers/"), "/json")
		deps := []string{}
		switch id {
		case summaries[0].ID:
			deps = []string{summaries[1].ID, "infra"}
		case summaries[1].ID:
			deps = []string{summaries[2].ID}
		case summaries[2].ID:
		default:
			t.Errorf("unexpected native inspection: %s", r.URL.Path)
		}
		require.NoError(t, json.MarshalWrite(w, &struct {
			ID           string `json:"Id"`
			Dependencies []string
		}{id, deps}))
	}))
	defer srv.Close()
	cs, err := ListContainers(t.Context(), types.DockerProviderConfig{URL: srv.URL})
	require.NoError(t, err)
	require.Len(t, cs, 6)
	require.EqualValues(t, 3, inspections.Load(), "explicit and Docker Compose dependencies need no native inspection")
	require.Equal(t, []string{"worker"}, cs[0].IdlewatcherConfig.DependsOn)
	require.Nil(t, cs[1].IdlewatcherConfig, "dependency services need no idle timeout")
	require.Equal(t, []string{"db"}, Dependencies(cs[1]), "recursive watcher discovery uses this accessor")
	require.Equal(t, []string{"manual"}, cs[3].IdlewatcherConfig.DependsOn)
	require.Equal(t, []string{"db:service_healthy"}, cs[4].IdlewatcherConfig.DependsOn)
	require.NotContains(t, cs[0].ActualLabels, LabelDependsOn, "native data must not become a fake Docker label")
}

func TestPodmanInspectionFailureKeepsContainers(t *testing.T) {
	for _, response := range []string{"denied", "malformed", "canceled"} {
		t.Run(response, func(t *testing.T) {
			var started atomic.Int32
			bothStarted := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/_ping") {
					w.Header().Set("API-Version", "1.47")
					return
				}
				if strings.HasSuffix(r.URL.Path, "/containers/json") {
					fmt.Fprint(w, `[{"Id":"a","Names":["/app"],"Labels":{"io.podman.compose.project":"p","proxy.idle_timeout":"1m"}},{"Id":"b","Names":["/worker"],"Labels":{"io.podman.compose.project":"p"}}]`)
					return
				}
				switch response {
				case "denied":
					w.WriteHeader(http.StatusForbidden)
				case "malformed":
					fmt.Fprint(w, `{"Id":`)
				case "canceled":
					if started.Add(1) == 2 {
						close(bothStarted)
					}
					<-r.Context().Done()
				}
			}))
			defer srv.Close()
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			if response == "canceled" {
				go func() {
					select {
					case <-bothStarted:
						cancel()
					case <-ctx.Done():
					}
				}()
			}
			cs, err := ListContainers(ctx, types.DockerProviderConfig{URL: srv.URL})
			require.NoError(t, err)
			require.Len(t, cs, 2)
			require.NotNil(t, cs[0].IdlewatcherConfig)
			require.Empty(t, Dependencies(cs[0]))
			if response == "canceled" {
				require.EqualValues(t, 2, started.Load(), "native inspections must run concurrently")
			}
		})
	}
}
