package provider

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/yusing/go-proxy/internal/docker"
	. "github.com/yusing/go-proxy/internal/utils/testing"
	"gopkg.in/yaml.v3"

	_ "embed"
)

//go:embed docker_labels.yaml
var testDockerLabelsYAML []byte

func TestParseDockerLabels(t *testing.T) {
	var provider DockerProvider

	labels := make(map[string]string)
	ExpectNoError(t, yaml.Unmarshal(testDockerLabelsYAML, &labels))

	routes, err := provider.routesFromContainerLabels(
		docker.FromDocker(&types.Container{
			Names:  []string{"container"},
			Labels: labels,
			State:  "running",
			Ports: []types.Port{
				{Type: "tcp", PrivatePort: 1234, PublicPort: 1234},
			},
		}, "/var/run/docker.sock"),
	)
	ExpectNoError(t, err)
	ExpectTrue(t, routes.Contains("app"))
	ExpectTrue(t, routes.Contains("app1"))
}
