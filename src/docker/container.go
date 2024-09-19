package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	U "github.com/yusing/go-proxy/utils"
)

type ProxyProperties struct {
	DockerHost    string   `yaml:"docker_host" json:"docker_host"`
	ContainerName string   `yaml:"container_name" json:"container_name"`
	ImageName     string   `yaml:"image_name" json:"image_name"`
	Aliases       []string `yaml:"aliases" json:"aliases"`
	IsExcluded    bool     `yaml:"is_excluded" json:"is_excluded"`
	FirstPort     string   `yaml:"first_port" json:"first_port"`
	IdleTimeout   string   `yaml:"idle_timeout" json:"idle_timeout"`
	WakeTimeout   string   `yaml:"wake_timeout" json:"wake_timeout"`
	StopMethod    string   `yaml:"stop_method" json:"stop_method"`
	StopTimeout   string   `yaml:"stop_timeout" json:"stop_timeout"` // stop_method = "stop" only
	StopSignal    string   `yaml:"stop_signal" json:"stop_signal"`   // stop_method = "stop" | "kill" only
}

type Container struct {
	*types.Container
	*ProxyProperties
}

func FromDocker(c *types.Container, dockerHost string) (res Container) {
	res.Container = c
	res.ProxyProperties = &ProxyProperties{
		DockerHost:    dockerHost,
		ContainerName: res.getName(),
		ImageName:     res.getImageName(),
		Aliases:       res.getAliases(),
		IsExcluded:    U.ParseBool(res.getDeleteLabel(LableExclude)),
		FirstPort:     res.firstPortOrEmpty(),
		IdleTimeout:   res.getDeleteLabel(LabelIdleTimeout),
		WakeTimeout:   res.getDeleteLabel(LabelWakeTimeout),
		StopMethod:    res.getDeleteLabel(LabelStopMethod),
		StopTimeout:   res.getDeleteLabel(LabelStopTimeout),
		StopSignal:    res.getDeleteLabel(LabelStopSignal),
	}
	return
}

func FromJson(json types.ContainerJSON, dockerHost string) Container {
	ports := make([]types.Port, 0)
	for k, bindings := range json.NetworkSettings.Ports {
		for _, v := range bindings {
			pubPort, _ := strconv.Atoi(v.HostPort)
			privPort, _ := strconv.Atoi(k.Port())
			ports = append(ports, types.Port{
				IP:          v.HostIP,
				PublicPort:  uint16(pubPort),
				PrivatePort: uint16(privPort),
			})
		}
	}
	return FromDocker(&types.Container{
		ID:     json.ID,
		Names:  []string{json.Name},
		Image:  json.Image,
		Ports:  ports,
		Labels: json.Config.Labels,
		State:  json.State.Status,
		Status: json.State.Status,
	}, dockerHost)
}

func (c Container) getDeleteLabel(label string) string {
	if l, ok := c.Labels[label]; ok {
		delete(c.Labels, label)
		return l
	}
	return ""
}

func (c Container) getAliases() []string {
	if l := c.getDeleteLabel(LableAliases); l != "" {
		return U.CommaSeperatedList(l)
	} else {
		return []string{c.getName()}
	}
}

func (c Container) getName() string {
	return strings.TrimPrefix(c.Names[0], "/")
}

func (c Container) getImageName() string {
	colonSep := strings.Split(c.Image, ":")
	slashSep := strings.Split(colonSep[len(colonSep)-1], "/")
	return slashSep[len(slashSep)-1]
}

func (c Container) firstPortOrEmpty() string {
	if len(c.Ports) == 0 {
		return ""
	}
	for _, p := range c.Ports {
		if p.PublicPort != 0 {
			return fmt.Sprint(p.PublicPort)
		}
	}
	return ""
}
