package provider

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/route"
	U "github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"github.com/yusing/go-proxy/internal/watcher"
	"gopkg.in/yaml.v3"
)

type DockerProvider struct {
	name, dockerHost string
	l                zerolog.Logger
}

const (
	aliasRefPrefix    = '#'
	aliasRefPrefixAlt = '$'
)

var ErrAliasRefIndexOutOfRange = E.New("index out of range")

func DockerProviderImpl(name, dockerHost string) (ProviderImpl, error) {
	if dockerHost == common.DockerHostFromEnv {
		dockerHost = common.GetEnvString("DOCKER_HOST", client.DefaultDockerHost)
	}
	return &DockerProvider{
		name,
		dockerHost,
		logger.With().Str("type", "docker").Str("name", name).Logger(),
	}, nil
}

func (p *DockerProvider) String() string {
	return "docker@" + p.name
}

func (p *DockerProvider) ShortName() string {
	return p.name
}

func (p *DockerProvider) IsExplicitOnly() bool {
	return p.name[len(p.name)-1] == '!'
}

func (p *DockerProvider) Logger() *zerolog.Logger {
	return &p.l
}

func (p *DockerProvider) NewWatcher() watcher.Watcher {
	return watcher.NewDockerWatcher(p.dockerHost)
}

func (p *DockerProvider) loadRoutesImpl() (route.Routes, E.Error) {
	routes := route.NewRoutes()
	entries := route.NewProxyEntries()

	containers, err := docker.ListContainers(p.dockerHost)
	if err != nil {
		return routes, E.From(err)
	}

	errs := E.NewBuilder("")

	for _, c := range containers {
		container := docker.FromDocker(&c, p.dockerHost)
		if container.IsExcluded {
			continue
		}

		newEntries, err := p.entriesFromContainerLabels(container)
		if err != nil {
			errs.Add(err.Subject(container.ContainerName))
		}
		// although err is not nil
		// there may be some valid entries in `en`
		dups := entries.MergeFrom(newEntries)
		// add the duplicate proxy entries to the error
		dups.RangeAll(func(k string, v *route.RawEntry) {
			errs.Addf("duplicated alias %s", k)
		})
	}

	routes, err = route.FromEntries(entries)
	errs.Add(err)

	return routes, errs.Error()
}

func (p *DockerProvider) shouldIgnore(container *docker.Container) bool {
	return container.IsExcluded ||
		!container.IsExplicit && p.IsExplicitOnly() ||
		!container.IsExplicit && container.IsDatabase ||
		strings.HasSuffix(container.ContainerName, "-old")
}

// Returns a list of proxy entries for a container.
// Always non-nil.
func (p *DockerProvider) entriesFromContainerLabels(container *docker.Container) (entries route.RawEntries, _ E.Error) {
	entries = route.NewProxyEntries()

	if p.shouldIgnore(container) {
		return
	}

	// init entries map for all aliases
	for _, a := range container.Aliases {
		entries.Store(a, &route.RawEntry{
			Alias:     a,
			Container: container,
		})
	}

	errs := E.NewBuilder("label errors")

	m, err := docker.ParseLabels(container.Labels)
	errs.Add(err)

	var wildcardProps docker.LabelMap

	for alias, entryMapAny := range m {
		if len(alias) == 0 {
			errs.Add(E.New("empty alias"))
			continue
		}

		entryMap, ok := entryMapAny.(docker.LabelMap)
		if !ok {
			// try to deserialize to map
			entryMap = make(docker.LabelMap)
			yamlStr, ok := entryMapAny.(string)
			if !ok {
				// should not happen
				panic(fmt.Errorf("invalid entry map type %T", entryMapAny))
			}
			if err := yaml.Unmarshal([]byte(yamlStr), &entryMap); err != nil {
				errs.Add(E.From(err).Subject(alias))
				continue
			}
		}

		if alias == docker.WildcardAlias {
			wildcardProps = entryMap
			continue
		}

		// check if it is an alias reference
		switch alias[0] {
		case aliasRefPrefix, aliasRefPrefixAlt:
			index, err := strutils.Atoi(alias[1:])
			if err != nil {
				errs.Add(err)
				break
			}
			if index < 1 || index > len(container.Aliases) {
				errs.Add(ErrAliasRefIndexOutOfRange.Subject(strconv.Itoa(index)))
				break
			}
			alias = container.Aliases[index-1]
		}

		// init entry if not exist
		en, ok := entries.Load(alias)
		if !ok {
			en = &route.RawEntry{
				Alias:     alias,
				Container: container,
			}
			entries.Store(alias, en)
		}

		// deserialize map into entry object
		err := U.Deserialize(entryMap, en)
		if err != nil {
			errs.Add(err.Subject(alias))
		} else {
			entries.Store(alias, en)
		}
	}
	if wildcardProps != nil {
		entries.Range(func(alias string, re *route.RawEntry) bool {
			if err := U.Deserialize(wildcardProps, re); err != nil {
				errs.Add(err.Subject(docker.WildcardAlias))
				return false
			}
			return true
		})
	}

	return entries, errs.Error()
}
