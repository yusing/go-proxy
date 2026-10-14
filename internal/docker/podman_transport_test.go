package docker

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/internal/types"
)

func TestListContainersPodmanUnixSocket(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "podman.sock")
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)
	appID, dbID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	var inspections atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.Header().Set("API-Version", "1.47")
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			fmt.Fprintf(w, `[{"Id":%q,"Names":["/stack_app_1"],"Labels":{"io.podman.compose.project":"stack","com.docker.compose.project":"stack","com.docker.compose.service":"app","proxy.idle_timeout":"1m"}},{"Id":%q,"Names":["/stack_db_1"],"Labels":{"com.docker.compose.project":"stack","com.docker.compose.service":"db"}}]`, appID, dbID)
		case r.URL.Path == "/v4.0.0/libpod/containers/"+appID+"/json":
			inspections.Add(1)
			fmt.Fprintf(w, `{"Id":%q,"Dependencies":[%q]}`, appID, dbID)
		default:
			t.Errorf("unexpected Podman request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	cs, err := ListContainers(t.Context(), types.DockerProviderConfig{URL: "unix://" + sock})
	require.NoError(t, err)
	require.Len(t, cs, 2)
	require.EqualValues(t, 1, inspections.Load())
	require.NotNil(t, cs[0].IdlewatcherConfig)
	require.Equal(t, []string{"db"}, cs[0].IdlewatcherConfig.DependsOn)
	require.Equal(t, []string{"db"}, Dependencies(cs[0]))
	require.Nil(t, cs[1].IdlewatcherConfig)
}
