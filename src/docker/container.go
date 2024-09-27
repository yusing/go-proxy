package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	U "github.com/yusing/go-proxy/utils"
)

type Container struct {
	*types.Container
	*ProxyProperties
}

func FromDocker(c *types.Container, dockerHost string) (res Container) {
	res.Container = c
	res.ProxyProperties = &ProxyProperties{
		DockerHost:         dockerHost,
		ContainerName:      res.getName(),
		ImageName:          res.getImageName(),
		PublicPortMapping:  res.getPublicPortMapping(),
		PrivatePortMapping: res.getPrivatePortMapping(),
		NetworkMode:        c.HostConfig.NetworkMode,
		Aliases:            res.getAliases(),
		IsExcluded:         U.ParseBool(res.getDeleteLabel(LabelExclude)),
		IdleTimeout:        res.getDeleteLabel(LabelIdleTimeout),
		WakeTimeout:        res.getDeleteLabel(LabelWakeTimeout),
		StopMethod:         res.getDeleteLabel(LabelStopMethod),
		StopTimeout:        res.getDeleteLabel(LabelStopTimeout),
		StopSignal:         res.getDeleteLabel(LabelStopSignal),
		Running:            c.Status == "running" || c.State == "running",
	}
	return
}

func FromJson(json types.ContainerJSON, dockerHost string) Container {
	ports := make([]types.Port, 0)
	for k, bindings := range json.NetworkSettings.Ports {
		for _, v := range bindings {
			pubPort, _ := strconv.ParseUint(v.HostPort, 10, 16)
			privPort, _ := strconv.ParseUint(k.Port(), 10, 16)
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
	if l := c.getDeleteLabel(LabelAliases); l != "" {
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

func (c Container) getPublicPortMapping() PortMapping {
	res := make(PortMapping)
	for _, v := range c.Ports {
		if v.PublicPort == 0 {
			continue
		}
		res[fmt.Sprint(v.PublicPort)] = v
	}
	return res
}

func (c Container) getPrivatePortMapping() PortMapping {
	res := make(PortMapping)
	for _, v := range c.Ports {
		res[fmt.Sprint(v.PrivatePort)] = v
	}
	return res
}
