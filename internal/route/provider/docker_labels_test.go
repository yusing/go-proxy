package provider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/internal/docker"
	"github.com/yusing/godoxy/internal/types"
	expect "github.com/yusing/goutils/testing"

	_ "embed"
)

//go:embed fixtures/docker_labels.yaml
var testDockerLabelsYAML []byte

func TestParseDockerLabels(t *testing.T) {
	var provider DockerProvider

	labels := make(map[string]string)
	expect.NoError(t, yaml.Unmarshal(testDockerLabelsYAML, &labels))

	routes, err := provider.routesFromContainerLabels(
		docker.FromDocker(t.Context(), &container.Summary{
			Names:  []string{"container"},
			Labels: labels,
			State:  "running",
			Ports: []container.Port{
				{Type: "tcp", PrivatePort: 1234, PublicPort: 1234},
			},
		}, types.DockerProviderConfig{URL: "unix:///var/run/docker.sock"}),
	)
	expect.NoError(t, err)
	expect.True(t, routes.Contains("app"))
	expect.True(t, routes.Contains("app1"))
}

func TestDockerLabelsCORS(t *testing.T) {
	const config = `port: 8080
rules: |
  method OPTIONS &
  header Origin &
  header Access-Control-Request-Method {
    set resp_header Access-Control-Allow-Origin $header(Origin)
    set resp_header Access-Control-Allow-Methods GET,POST,PUT,PATCH,DELETE,OPTIONS
    set resp_header Access-Control-Allow-Headers $header(Access-Control-Request-Headers)
    set resp_header Access-Control-Allow-Credentials true
  }
  method OPTIONS &
  header Origin &
  header Access-Control-Request-Method {
    error 204 ""
  }
  header Origin {
    set resp_header Access-Control-Allow-Origin $header(Origin)
    set resp_header Access-Control-Allow-Credentials true
  }
`
	var provider DockerProvider
	load := func(config string) *docker.Container {
		return docker.FromDocker(t.Context(), &container.Summary{
			Names: []string{"headscale"},
			Labels: map[string]string{
				"proxy.aliases":   "headscale",
				"proxy.headscale": config,
			},
			State: "running",
		}, types.DockerProviderConfig{URL: "unix:///var/run/docker.sock"})
	}
	t.Run("folded rules", func(t *testing.T) {
		_, err := provider.routesFromContainerLabels(load(strings.Replace(config, "rules: |", "rules:", 1)))
		require.ErrorContains(t, err, "expect 2 args")
	})
	routes, err := provider.routesFromContainerLabels(load(config))
	require.NoError(t, err)
	for _, tc := range []struct {
		name, method, origin, requestMethod string
		preflight                           bool
	}{
		{"preflight", http.MethodOptions, "https://headplane.example.com", "POST", true},
		{"actual request", http.MethodGet, "https://headplane.example.com", "", false},
		{"non-CORS request", http.MethodGet, "", "", false},
		{"ordinary OPTIONS", http.MethodOptions, "https://headplane.example.com", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstreamCalled := false
			handler := routes["headscale"].Rules.BuildHandler(func(w http.ResponseWriter, r *http.Request) {
				upstreamCalled = true
				_, _ = w.Write([]byte("upstream"))
			})
			req := httptest.NewRequest(tc.method, "/api/v1/node", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.requestMethod != "" {
				req.Header.Set("Access-Control-Request-Method", tc.requestMethod)
				req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, !tc.preflight, upstreamCalled)
			assert.Equal(t, tc.origin, w.Header().Get("Access-Control-Allow-Origin"))
			if tc.origin != "" {
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
			} else {
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
			}
			if tc.preflight {
				assert.Equal(t, http.StatusNoContent, w.Code)
				assert.Empty(t, w.Body.String())
				assert.Equal(t, "GET,POST,PUT,PATCH,DELETE,OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Authorization,Content-Type", w.Header().Get("Access-Control-Allow-Headers"))
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, "upstream", w.Body.String())
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Methods"))
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Headers"))
			}
		})
	}
}
