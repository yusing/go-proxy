package provider

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	"github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"github.com/yusing/go-proxy/internal/watcher"
)

type DockerProvider struct {
	name, dockerHost string
	ExplicitOnly     bool
	l                zerolog.Logger
}

var (
	AliasRefRegex    = regexp.MustCompile(`#\d+`)
	AliasRefRegexOld = regexp.MustCompile(`\$\d+`)

	ErrAliasRefIndexOutOfRange = E.New("index out of range")
)

func DockerProviderImpl(name, dockerHost string, explicitOnly bool) (ProviderImpl, error) {
	if dockerHost == common.DockerHostFromEnv {
		dockerHost = common.GetEnv("DOCKER_HOST", client.DefaultDockerHost)
	}
	return &DockerProvider{
		name,
		dockerHost,
		explicitOnly,
		logger.With().Str("type", "docker").Str("name", name).Logger(),
	}, nil
}

func (p *DockerProvider) String() string {
	return "docker@" + p.name
}

func (p *DockerProvider) Logger() *zerolog.Logger {
	return &p.l
}

func (p *DockerProvider) NewWatcher() watcher.Watcher {
	return watcher.NewDockerWatcher(p.dockerHost)
}

func (p *DockerProvider) loadRoutesImpl() (route.Routes, E.Error) {
	routes := route.NewRoutes()
	entries := entry.NewProxyEntries()

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
		dups.RangeAll(func(k string, v *entry.RawEntry) {
			errs.Addf("duplicated alias %s", k)
		})
	}

	routes, err = route.FromEntries(entries)
	errs.Add(err)

	return routes, errs.Error()
}

func (p *DockerProvider) shouldIgnore(container *docker.Container) bool {
	return container.IsExcluded ||
		!container.IsExplicit && p.ExplicitOnly ||
		!container.IsExplicit && container.IsDatabase ||
		strings.HasSuffix(container.ContainerName, "-old")
}

// Returns a list of proxy entries for a container.
// Always non-nil.
func (p *DockerProvider) entriesFromContainerLabels(container *docker.Container) (entries entry.RawEntries, _ E.Error) {
	entries = entry.NewProxyEntries()

	if p.shouldIgnore(container) {
		return
	}

	// init entries map for all aliases
	for _, a := range container.Aliases {
		entries.Store(a, &entry.RawEntry{
			Alias:     a,
			Container: container,
		})
	}

	errs := E.NewBuilder("label errors")
	for key, val := range container.Labels {
		errs.Add(p.applyLabel(container, entries, key, val))
	}

	// remove all entries that failed to fill in missing fields
	entries.RangeAll(func(_ string, re *entry.RawEntry) {
		re.FillMissingFields()
	})

	return entries, errs.Error()
}

func (p *DockerProvider) applyLabel(container *docker.Container, entries entry.RawEntries, key, val string) E.Error {
	lbl := docker.ParseLabel(key, val)
	if lbl.Namespace != docker.NSProxy {
		return nil
	}
	if lbl.Target == docker.WildcardAlias {
		// apply label for all aliases
		labelErrs := entries.CollectErrors(func(a string, e *entry.RawEntry) error {
			return docker.ApplyLabel(e, lbl)
		})
		if err := E.Join(labelErrs...); err != nil {
			return err.Subject(lbl.Target)
		}
		return nil
	}

	refErrs := E.NewBuilder("alias ref errors")
	replaceIndexRef := func(ref string) string {
		index, err := strutils.Atoi(ref[1:])
		if err != nil {
			refErrs.Add(err)
			return ref
		}
		if index < 1 || index > len(container.Aliases) {
			refErrs.Add(ErrAliasRefIndexOutOfRange.Subject(strconv.Itoa(index)))
			return ref
		}
		return container.Aliases[index-1]
	}

	lbl.Target = AliasRefRegex.ReplaceAllStringFunc(lbl.Target, replaceIndexRef)
	lbl.Target = AliasRefRegexOld.ReplaceAllStringFunc(lbl.Target, func(ref string) string {
		p.l.Warn().Msgf("%q should now be %q, old syntax will be removed in a future version", lbl, strings.ReplaceAll(lbl.String(), "$", "#"))
		return replaceIndexRef(ref)
	})
	if refErrs.HasError() {
		return refErrs.Error().Subject(lbl.String())
	}

	en, ok := entries.Load(lbl.Target)
	if !ok {
		en = &entry.RawEntry{
			Alias:     lbl.Target,
			Container: container,
		}
		entries.Store(lbl.Target, en)
	}

	return docker.ApplyLabel(en, lbl)
}
