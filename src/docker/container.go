package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	U "github.com/yusing/go-proxy/utils"
)

type ProxyProperties struct {
	DockerHost    string   `yaml:"-" json:"docker_host"`
	ContainerName string   `yaml:"-" json:"container_name"`
	ImageName     string   `yaml:"-" json:"image_name"`
	Aliases       []string `yaml:"-" json:"aliases"`
	IsExcluded    bool     `yaml:"-" json:"is_excluded"`
	FirstPort     string   `yaml:"-" json:"first_port"`
	IdleTimeout   string   `yaml:"-" json:"idle_timeout"`
	WakeTimeout   string   `yaml:"-" json:"wake_timeout"`
	StopMethod    string   `yaml:"-" json:"stop_method"`
	StopTimeout   string   `yaml:"-" json:"stop_timeout"` // stop_method = "stop" only
	StopSignal    string   `yaml:"-" json:"stop_signal"`  // stop_method = "stop" | "kill" only
	Running       bool     `yaml:"-" json:"running"`
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
		Running:       c.Status == "running" || c.State == "running",
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
	slashSep := strings.Split(colonSep[0], "/")
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
