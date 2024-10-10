package docker

import "github.com/docker/docker/api/types"

type (
	PortMapping     = map[string]types.Port
	ProxyProperties struct {
		DockerHost         string      `json:"docker_host" yaml:"-"`
		ContainerName      string      `json:"container_name" yaml:"-"`
		ContainerID        string      `json:"container_id" yaml:"-"`
		ImageName          string      `json:"image_name" yaml:"-"`
		PublicPortMapping  PortMapping `json:"public_ports" yaml:"-"`  // non-zero publicPort:types.Port
		PrivatePortMapping PortMapping `json:"private_ports" yaml:"-"` // privatePort:types.Port
		NetworkMode        string      `json:"network_mode" yaml:"-"`

		Aliases     []string `json:"aliases" yaml:"-"`
		IsExcluded  bool     `json:"is_excluded" yaml:"-"`
		IsExplicit  bool     `json:"is_explicit" yaml:"-"`
		IsDatabase  bool     `json:"is_database" yaml:"-"`
		IdleTimeout string   `json:"idle_timeout" yaml:"-"`
		WakeTimeout string   `json:"wake_timeout" yaml:"-"`
		StopMethod  string   `json:"stop_method" yaml:"-"`
		StopTimeout string   `json:"stop_timeout" yaml:"-"` // stop_method = "stop" only
		StopSignal  string   `json:"stop_signal" yaml:"-"`  // stop_method = "stop" | "kill" only
		Running     bool     `json:"running" yaml:"-"`
	}
)
