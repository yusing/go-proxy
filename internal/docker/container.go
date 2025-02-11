package docker

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/yusing/go-proxy/agent/pkg/agent"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/logging"
	U "github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	PortMapping = map[int]types.Port
	Container   struct {
		_ U.NoCopy

		DockerHost    string `json:"docker_host"`
		ContainerName string `json:"container_name"`
		ContainerID   string `json:"container_id"`
		ImageName     string `json:"image_name"`

		Agent *agent.AgentConfig `json:"agent"`

		Labels map[string]string `json:"-"`

		PublicPortMapping  PortMapping `json:"public_ports"`  // non-zero publicPort:types.Port
		PrivatePortMapping PortMapping `json:"private_ports"` // privatePort:types.Port
		PublicHostname     string      `json:"public_hostname"`
		PrivateHostname    string      `json:"private_hostname"`

		Aliases       []string `json:"aliases"`
		IsExcluded    bool     `json:"is_excluded"`
		IsExplicit    bool     `json:"is_explicit"`
		IsDatabase    bool     `json:"is_database"`
		IdleTimeout   string   `json:"idle_timeout,omitempty"`
		WakeTimeout   string   `json:"wake_timeout,omitempty"`
		StopMethod    string   `json:"stop_method,omitempty"`
		StopTimeout   string   `json:"stop_timeout,omitempty"` // stop_method = "stop" only
		StopSignal    string   `json:"stop_signal,omitempty"`  // stop_method = "stop" | "kill" only
		StartEndpoint string   `json:"start_endpoint,omitempty"`
		Running       bool     `json:"running"`
	}
)

var DummyContainer = new(Container)

func FromDocker(c *types.Container, dockerHost string) (res *Container) {
	isExplicit := false
	helper := containerHelper{c}
	for lbl := range c.Labels {
		if strings.HasPrefix(lbl, NSProxy+".") {
			isExplicit = true
		} else {
			delete(c.Labels, lbl)
		}
	}
	res = &Container{
		DockerHost:    dockerHost,
		ContainerName: helper.getName(),
		ContainerID:   c.ID,
		ImageName:     helper.getImageName(),

		Labels: c.Labels,

		PublicPortMapping:  helper.getPublicPortMapping(),
		PrivatePortMapping: helper.getPrivatePortMapping(),

		Aliases:       helper.getAliases(),
		IsExcluded:    strutils.ParseBool(helper.getDeleteLabel(LabelExclude)),
		IsExplicit:    isExplicit,
		IsDatabase:    helper.isDatabase(),
		IdleTimeout:   helper.getDeleteLabel(LabelIdleTimeout),
		WakeTimeout:   helper.getDeleteLabel(LabelWakeTimeout),
		StopMethod:    helper.getDeleteLabel(LabelStopMethod),
		StopTimeout:   helper.getDeleteLabel(LabelStopTimeout),
		StopSignal:    helper.getDeleteLabel(LabelStopSignal),
		StartEndpoint: helper.getDeleteLabel(LabelStartEndpoint),
		Running:       c.Status == "running" || c.State == "running",
	}

	if agent.IsDockerHostAgent(dockerHost) {
		var ok bool
		res.Agent, ok = config.GetInstance().GetAgent(dockerHost)
		if !ok {
			logging.Error().Msgf("agent %q not found", dockerHost)
		}
	}

	res.setPrivateHostname(helper)
	res.setPublicHostname()
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
	return cont
}

func (c *Container) setPublicHostname() {
	if !c.Running {
		return
	}
	if strings.HasPrefix(c.DockerHost, "unix://") {
		c.PublicHostname = "127.0.0.1"
		return
	}
	url, err := url.Parse(c.DockerHost)
	if err != nil {
		logging.Err(err).Msgf("invalid docker host %q, falling back to 127.0.0.1", c.DockerHost)
		c.PublicHostname = "127.0.0.1"
		return
	}
	c.PublicHostname = url.Hostname()
}

func (c *Container) setPrivateHostname(helper containerHelper) {
	if !strings.HasPrefix(c.DockerHost, "unix://") && c.Agent == nil {
		return
	}
	if helper.NetworkSettings == nil {
		return
	}
	for _, v := range helper.NetworkSettings.Networks {
		if v.IPAddress == "" {
			continue
		}
		c.PrivateHostname = v.IPAddress
		return
	}
}
