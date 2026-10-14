package idlewatcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/internal/docker"
	idlewatchertypes "github.com/yusing/godoxy/internal/idlewatcher/runtime"
	"github.com/yusing/godoxy/internal/types"
)

func TestPodmanComposeRecursiveWakeAndStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("API-Version", "1.47")
			return
		}
		if strings.HasSuffix(r.URL.Path, "/containers/json") {
			fmt.Fprint(w, `[
                {"Id":"app","Names":["/stack_app_1"],"Labels":{"com.docker.compose.project":"stack","com.docker.compose.service":"app","io.podman.compose.project":"stack","proxy.idle_timeout":"1h"}},
                {"Id":"worker","Names":["/stack_worker_1"],"Labels":{"com.docker.compose.project":"stack","com.docker.compose.service":"worker","io.podman.compose.project":"stack"}},
                {"Id":"db","Names":["/stack_db_1"],"Labels":{"com.docker.compose.project":"stack","com.docker.compose.service":"db","io.podman.compose.project":"stack"}}
            ]`)
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v4.0.0/libpod/containers/"), "/json")
		deps := map[string]string{"app": `["worker"]`, "worker": `["db"]`, "db": `[]`}
		fmt.Fprintf(w, `{"Id":%q,"Dependencies":%s}`, id, deps[id])
	}))
	defer srv.Close()
	containers, err := docker.ListContainers(t.Context(), types.DockerProviderConfig{URL: srv.URL})
	require.NoError(t, err)

	_, parent, mainRoute, closes := newDependencyReloadTest(t, "podman-fixture", nil)
	routeProvider := mainRoute.provider.(*idlewatcherTestRouteProvider)
	var mu sync.Mutex
	var actions []string
	newDockerProvider = func(_ context.Context, _ types.DockerProviderConfig, id string) (idlewatchertypes.Provider, error) {
		return &podmanLifecycleProvider{
			idlewatcherTestRuntimeProvider: idlewatcherTestRuntimeProvider{closes: closes},
			record:                         func(action string) { mu.Lock(); defer mu.Unlock(); actions = append(actions, action+":"+id) },
		}, nil
	}
	for _, c := range containers {
		r := newIdlewatcherTestRoute(docker.DockerComposeService(c), routeProvider, c.IdlewatcherConfig)
		r.container = c
		routeProvider.routes[r.name] = r
	}
	app := routeProvider.routes["app"].(*idlewatcherTestRoute)
	w, err := NewWatcher(parent, app, app.cfg)
	require.NoError(t, err)
	require.NoError(t, w.Wake(t.Context()))
	mu.Lock()
	require.Equal(t, []string{"start:db", "start:worker", "start:app"}, actions)
	actions = nil
	mu.Unlock()

	// Model the runtime's start events before requesting group sleep.
	for _, dep := range w.dependencies() {
		dep.storeState(&containerState{status: idlewatchertypes.ContainerStatusRunning})
	}
	w.storeState(&containerState{status: idlewatchertypes.ContainerStatusRunning})
	require.NoError(t, w.stopByMethod())
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"stop:app", "stop:worker", "stop:db"}, actions)
}

type podmanLifecycleProvider struct {
	idlewatcherTestRuntimeProvider
	record func(string)
}

func (p *podmanLifecycleProvider) ContainerStart(context.Context) error {
	p.record("start")
	return nil
}
func (p *podmanLifecycleProvider) ContainerStop(context.Context, idlewatchertypes.Signal, int) error {
	p.record("stop")
	return nil
}
