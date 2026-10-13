package docker

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/internal/types"
)

const (
	podmanAppID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	podmanDBID  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	podmanWebID = "ccccccccccccccccbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	// podmanUnknownID is in the libpod inspect output but not in the list
	// snapshot, so name resolution has nothing for it.
	podmanUnknownID = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

// podmanAPIStub serves the two endpoints podmanDependencies reads: the native
// libpod inspect and the Docker-compatible list used to resolve container IDs.
func podmanAPIStub(t *testing.T) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v4.0.0/libpod/containers/"+podmanAppID+"/json", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"Id":"`+podmanAppID+`","Dependencies":["`+podmanDBID+`","`+podmanWebID+`","`+podmanUnknownID+`"]}`)
	})
	mux.HandleFunc("/containers/json", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[
			{"Id":"`+podmanDBID+`","Names":["/repro_db_1"],"Labels":{"com.docker.compose.service":"db"}},
			{"Id":"`+podmanWebID+`","Names":["/repro_web_1"],"Labels":{"io.podman.compose.service":"web"}}
		]`)
	})
	return mux
}

func TestPodmanDependenciesSkipsUnsupportedHosts(t *testing.T) {
	tests := []struct {
		name string
		c    *Container
	}{
		{
			// Docker Compose reports depends_on in a label.
			name: "no podman-compose label",
			c: &Container{
				DockerCfg:   types.DockerProviderConfig{URL: "unix:///run/docker.sock"},
				ContainerID: podmanAppID,
			},
		},
		{
			name: "empty container id",
			c: &Container{
				DockerCfg:    types.DockerProviderConfig{URL: "unix:///run/podman/podman.sock"},
				ActualLabels: map[string]string{labelPodmanComposeProject: "repro"},
			},
		},
		{
			name: "agent not resolved",
			c: &Container{
				DockerCfg:    types.DockerProviderConfig{URL: "agent://10.0.0.1:8181"},
				ContainerID:  podmanAppID,
				ActualLabels: map[string]string{labelPodmanComposeProject: "repro"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Nil(t, podmanDependencies(t.Context(), tc.c))
		})
	}
}

func TestFetchPodmanDependencies(t *testing.T) {
	srv := httptest.NewServer(podmanAPIStub(t))
	defer srv.Close()

	got, err := fetchPodmanDependencies(t.Context(), srv.Client(), srv.URL, podmanAppID)
	require.NoError(t, err)
	// The snapshot returns service names where the compose provider reports
	// them, container names otherwise, and drops the unknown ID.
	require.Equal(t, []string{"db", "web"}, got)
}

func TestFetchPodmanDependenciesNotPodman(t *testing.T) {
	srv := httptest.NewServer(http.NewServeMux()) // Docker has no libpod API
	defer srv.Close()

	got, err := fetchPodmanDependencies(t.Context(), srv.Client(), srv.URL, podmanAppID)
	require.Error(t, err)
	require.Nil(t, got)
}

func TestFetchPodmanDependenciesInvalidID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v4.0.0/libpod/containers/"+podmanAppID+"/json", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"Id":"`+podmanAppID+`","Dependencies":["not-a-container-id"]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	got, err := fetchPodmanDependencies(t.Context(), srv.Client(), srv.URL, podmanAppID)
	require.Error(t, err)
	require.Nil(t, got)
}

// TestPodmanDependenciesUnixSocket checks that the native libpod API is
// reachable over the same unix socket as the Docker-compatible API.
func TestPodmanDependenciesUnixSocket(t *testing.T) {
	socketPath := serveStubOnUnixSocket(t, podmanAPIStub(t))

	c := &Container{
		DockerCfg:    types.DockerProviderConfig{URL: "unix://" + socketPath},
		ContainerID:  podmanAppID,
		ActualLabels: map[string]string{labelPodmanComposeProject: "repro"},
	}
	require.Equal(t, []string{"db", "web"}, podmanDependencies(t.Context(), c))
}

func serveStubOnUnixSocket(t *testing.T, handler http.Handler) string {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "podman.sock")
	ln, err := net.Listen("unix", socketPath)
	require.NoError(t, err)

	srv := &http.Server{Handler: handler}
	go func() {
		_ = srv.Serve(ln)
	}()
	t.Cleanup(func() {
		srv.Close()
		ln.Close()
	})

	return socketPath
}
