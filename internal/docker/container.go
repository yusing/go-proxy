package docker

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
	PortMapping = map[string]types.Port
	Container   struct {
		_ U.NoCopy

		DockerHost    string `json:"docker_host" yaml:"-"`
		ContainerName string `json:"container_name" yaml:"-"`
		ContainerID   string `json:"container_id" yaml:"-"`
		ImageName     string `json:"image_name" yaml:"-"`

		Labels map[string]string `json:"labels" yaml:"-"`

		PublicPortMapping  PortMapping `json:"public_ports" yaml:"-"`  // non-zero publicPort:types.Port
		PrivatePortMapping PortMapping `json:"private_ports" yaml:"-"` // privatePort:types.Port
		PublicIP           string      `json:"public_ip" yaml:"-"`
		PrivateIP          string      `json:"private_ip" yaml:"-"`
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

var DummyContainer = new(Container)

func FromDocker(c *types.Container, dockerHost string) (res *Container) {
	isExplicit := c.Labels[LabelAliases] != ""
	helper := containerHelper{c}
	res = &Container{
		DockerHost:    dockerHost,
		ContainerName: helper.getName(),
		ContainerID:   c.ID,
		ImageName:     helper.getImageName(),

		Labels: c.Labels,

		PublicPortMapping:  helper.getPublicPortMapping(),
		PrivatePortMapping: helper.getPrivatePortMapping(),
		NetworkMode:        c.HostConfig.NetworkMode,

		Aliases:     helper.getAliases(),
		IsExcluded:  U.ParseBool(helper.getDeleteLabel(LabelExclude)),
		IsExplicit:  isExplicit,
		IsDatabase:  helper.isDatabase(),
		IdleTimeout: helper.getDeleteLabel(LabelIdleTimeout),
		WakeTimeout: helper.getDeleteLabel(LabelWakeTimeout),
		StopMethod:  helper.getDeleteLabel(LabelStopMethod),
		StopTimeout: helper.getDeleteLabel(LabelStopTimeout),
		StopSignal:  helper.getDeleteLabel(LabelStopSignal),
		Running:     c.Status == "running" || c.State == "running",
	}
	res.setPrivateIP(helper)
	res.setPublicIP()
	return
}

func FromJSON(json types.ContainerJSON, dockerHost string) *Container {
	ports := make([]types.Port, 0)
	for k, bindings := range json.NetworkSettings.Ports {
		privPortStr, proto := k.Port(), k.Proto()
		privPort, _ := strconv.ParseUint(privPortStr, 10, 16)
		ports = append(ports, types.Port{
			PrivatePort: uint16(privPort),
			Type:        proto,
		})
		for _, v := range bindings {
			pubPort, _ := strconv.ParseUint(v.HostPort, 10, 16)
			ports = append(ports, types.Port{
				IP:          v.HostIP,
				PublicPort:  uint16(pubPort),
				PrivatePort: uint16(privPort),
				Type:        proto,
			})
		}
	}
	cont := FromDocker(&types.Container{
		ID:     json.ID,
		Names:  []string{strings.TrimPrefix(json.Name, "/")},
		Image:  json.Image,
		Ports:  ports,
		Labels: json.Config.Labels,
		State:  json.State.Status,
		Status: json.State.Status,
		Mounts: json.Mounts,
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: json.NetworkSettings.Networks,
		},
	}, dockerHost)
	cont.NetworkMode = string(json.HostConfig.NetworkMode)
	return cont
}

func (c *Container) setPublicIP() {
	if !c.Running {
		return
	}
	if strings.HasPrefix(c.DockerHost, "unix://") {
		c.PublicIP = "127.0.0.1"
		return
	}
	url, err := url.Parse(c.DockerHost)
	if err != nil {
		logrus.Errorf("invalid docker host %q: %v\nfalling back to 127.0.0.1", c.DockerHost, err)
		c.PublicIP = "127.0.0.1"
		return
	}
	c.PublicIP = url.Hostname()
}

func (c *Container) setPrivateIP(helper containerHelper) {
	if !strings.HasPrefix(c.DockerHost, "unix://") {
		return
	}
	if helper.NetworkSettings == nil {
		return
	}
	for _, v := range helper.NetworkSettings.Networks {
		if v.IPAddress == "" {
			continue
		}
		c.PrivateIP = v.IPAddress
		return
	}
}
