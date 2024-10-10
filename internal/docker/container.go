package docker

import (
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	U "github.com/yusing/go-proxy/internal/utils"
)

type Container struct {
	*types.Container
	*ProxyProperties
}

func FromDocker(c *types.Container, dockerHost string) (res Container) {
	res.Container = c
	isExplicit := c.Labels[LabelAliases] != ""
	res.ProxyProperties = &ProxyProperties{
		DockerHost:         dockerHost,
		ContainerName:      res.getName(),
		ContainerID:        c.ID,
		ImageName:          res.getImageName(),
		PublicPortMapping:  res.getPublicPortMapping(),
		PrivatePortMapping: res.getPrivatePortMapping(),
		NetworkMode:        c.HostConfig.NetworkMode,
		Aliases:            res.getAliases(),
		IsExcluded:         U.ParseBool(res.getDeleteLabel(LabelExclude)),
		IsExplicit:         isExplicit,
		IsDatabase:         res.isDatabase(),
		IdleTimeout:        res.getDeleteLabel(LabelIdleTimeout),
		WakeTimeout:        res.getDeleteLabel(LabelWakeTimeout),
		StopMethod:         res.getDeleteLabel(LabelStopMethod),
		StopTimeout:        res.getDeleteLabel(LabelStopTimeout),
		StopSignal:         res.getDeleteLabel(LabelStopSignal),
		Running:            c.Status == "running" || c.State == "running",
	}
	return
}

func FromJSON(json types.ContainerJSON, dockerHost string) Container {
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
	cont := FromDocker(&types.Container{
		ID:     json.ID,
		Names:  []string{json.Name},
		Image:  json.Image,
		Ports:  ports,
		Labels: json.Config.Labels,
		State:  json.State.Status,
		Status: json.State.Status,
	}, dockerHost)
	cont.NetworkMode = string(json.HostConfig.NetworkMode)
	return cont
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
	}
	return []string{c.getName()}
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
		res[U.PortString(v.PublicPort)] = v
	}
	return res
}

func (c Container) getPrivatePortMapping() PortMapping {
	res := make(PortMapping)
	for _, v := range c.Ports {
		res[U.PortString(v.PrivatePort)] = v
	}
	return res
}

var databaseMPs = map[string]struct{}{
	"/var/lib/postgresql/data": {},
	"/var/lib/mysql":           {},
	"/var/lib/mongodb":         {},
	"/var/lib/mariadb":         {},
	"/var/lib/memcached":       {},
	"/var/lib/rabbitmq":        {},
}

var databasePrivPorts = map[uint16]struct{}{
	5432:  {}, // postgres
	3306:  {}, // mysql, mariadb
	6379:  {}, // redis
	11211: {}, // memcached
	27017: {}, // mongodb
}

func (c Container) isDatabase() bool {
	for _, m := range c.Container.Mounts {
		if _, ok := databaseMPs[m.Destination]; ok {
			return true
		}
	}

	for _, v := range c.Ports {
		if _, ok := databasePrivPorts[v.PrivatePort]; ok {
			return true
		}
	}
	return false
}
