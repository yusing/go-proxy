package docker

import (
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type containerHelper struct {
	*types.Container
}

// getDeleteLabel gets the value of a label and then deletes it from the container.
// If the label does not exist, an empty string is returned.
func (c containerHelper) getDeleteLabel(label string) string {
	if l, ok := c.Labels[label]; ok {
		delete(c.Labels, label)
		return l
	}
	return ""
}

func (c containerHelper) getAliases() []string {
	if l := c.getDeleteLabel(LabelAliases); l != "" {
		return strutils.CommaSeperatedList(l)
	}
	return []string{c.getName()}
}

func (c containerHelper) getName() string {
	return strings.TrimPrefix(c.Names[0], "/")
}

func (c containerHelper) getImageName() string {
	colonSep := strutils.SplitRune(c.Image, ':')
	slashSep := strutils.SplitRune(colonSep[0], '/')
	return slashSep[len(slashSep)-1]
}

func (c containerHelper) getPublicPortMapping() PortMapping {
	res := make(PortMapping)
	for _, v := range c.Ports {
		if v.PublicPort == 0 {
			continue
		}
		res[int(v.PublicPort)] = v
	}
	return res
}

func (c containerHelper) getPrivatePortMapping() PortMapping {
	res := make(PortMapping)
	for _, v := range c.Ports {
		res[int(v.PrivatePort)] = v
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

func (c containerHelper) isDatabase() bool {
	for _, m := range c.Mounts {
		if _, ok := databaseMPs[m.Destination]; ok {
			return true
		}
	}

	for _, v := range c.Ports {
		switch v.PrivatePort {
		// postgres, mysql or mariadb, redis, memcached, mongodb
		case 5432, 3306, 6379, 11211, 27017:
			return true
		}
	}
	return false
}
