package docker

import "github.com/docker/docker/api/types"

type PortMapping = map[string]types.Port
type ProxyProperties struct {
	DockerHost         string      `yaml:"-" json:"docker_host"`
	ContainerName      string      `yaml:"-" json:"container_name"`
	ContainerID        string      `yaml:"-" json:"container_id"`
	ImageName          string      `yaml:"-" json:"image_name"`
	PublicPortMapping  PortMapping `yaml:"-" json:"public_port_mapping"`  // non-zero publicPort:types.Port
	PrivatePortMapping PortMapping `yaml:"-" json:"private_port_mapping"` // privatePort:types.Port
	NetworkMode        string      `yaml:"-" json:"network_mode"`

	Aliases     []string `yaml:"-" json:"aliases"`
	IsExcluded  bool     `yaml:"-" json:"is_excluded"`
	IsExplicit  bool     `yaml:"-" json:"is_explicit"`
	IsDatabase  bool     `yaml:"-" json:"is_database"`
	IdleTimeout string   `yaml:"-" json:"idle_timeout"`
	WakeTimeout string   `yaml:"-" json:"wake_timeout"`
	StopMethod  string   `yaml:"-" json:"stop_method"`
	StopTimeout string   `yaml:"-" json:"stop_timeout"` // stop_method = "stop" only
	StopSignal  string   `yaml:"-" json:"stop_signal"`  // stop_method = "stop" | "kill" only
	Running     bool     `yaml:"-" json:"running"`
}
