package provider

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/common"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	R "github.com/yusing/go-proxy/internal/route"
	W "github.com/yusing/go-proxy/internal/watcher"
)

type DockerProvider struct {
	name, dockerHost string
	ExplicitOnly     bool
}

var (
	AliasRefRegex    = regexp.MustCompile(`#\d+`)
	AliasRefRegexOld = regexp.MustCompile(`\$\d+`)
)

func DockerProviderImpl(name, dockerHost string, explicitOnly bool) (ProviderImpl, E.Error) {
	if dockerHost == common.DockerHostFromEnv {
		dockerHost = common.GetEnv("DOCKER_HOST", client.DefaultDockerHost)
	}
	return &DockerProvider{name, dockerHost, explicitOnly}, nil
}

func (p *DockerProvider) String() string {
	return "docker@" + p.name
}

func (p *DockerProvider) NewWatcher() W.Watcher {
	return W.NewDockerWatcher(p.dockerHost)
}

func (p *DockerProvider) LoadRoutesImpl() (routes R.Routes, err E.Error) {
	routes = R.NewRoutes()
	entries := entry.NewProxyEntries()

	containers, err := D.ListContainers(p.dockerHost)
	if err != nil {
		return routes, err
	}

	errors := E.NewBuilder("errors in docker labels")

	for _, c := range containers {
		container := D.FromDocker(&c, p.dockerHost)
		if container.IsExcluded {
			continue
		}

		newEntries, err := p.entriesFromContainerLabels(container)
		if err != nil {
			errors.Add(err)
		}
		// although err is not nil
		// there may be some valid entries in `en`
		dups := entries.MergeFrom(newEntries)
		// add the duplicate proxy entries to the error
		dups.RangeAll(func(k string, v *entry.RawEntry) {
			errors.Addf("duplicate alias %s", k)
		})
	}

	routes, err = R.FromEntries(entries)
	errors.Add(err)

	return routes, errors.Build()
}

func (p *DockerProvider) shouldIgnore(container *D.Container) bool {
	return container.IsExcluded ||
		!container.IsExplicit && p.ExplicitOnly ||
		!container.IsExplicit && container.IsDatabase ||
		strings.HasSuffix(container.ContainerName, "-old")
}

// Returns a list of proxy entries for a container.
// Always non-nil.
func (p *DockerProvider) entriesFromContainerLabels(container *D.Container) (entries entry.RawEntries, _ E.Error) {
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

	errors := E.NewBuilder("failed to apply label")
	for key, val := range container.Labels {
		errors.Add(p.applyLabel(container, entries, key, val))
	}

	// remove all entries that failed to fill in missing fields
	entries.RangeAll(func(_ string, re *entry.RawEntry) {
		re.FillMissingFields()
	})

	return entries, errors.Build().Subject(container.ContainerName)
}

func (p *DockerProvider) applyLabel(container *D.Container, entries entry.RawEntries, key, val string) (res E.Error) {
	b := E.NewBuilder("errors in label %s", key)
	defer b.To(&res)

	refErr := E.NewBuilder("errors in alias references")
	replaceIndexRef := func(ref string) string {
		index, err := strconv.Atoi(ref[1:])
		if err != nil {
			refErr.Add(E.Invalid("integer", ref))
			return ref
		}
		if index < 1 || index > len(container.Aliases) {
			refErr.Add(E.OutOfRange("index", ref))
			return ref
		}
		return container.Aliases[index-1]
	}

	lbl, err := D.ParseLabel(key, val)
	if err != nil {
		b.Add(err.Subject(key))
	}
	if lbl.Namespace != D.NSProxy {
		return
	}
	if lbl.Target == D.WildcardAlias {
		// apply label for all aliases
		entries.RangeAll(func(a string, e *entry.RawEntry) {
			if err = D.ApplyLabel(e, lbl); err != nil {
				b.Add(err)
			}
		})
	} else {
		lbl.Target = AliasRefRegex.ReplaceAllStringFunc(lbl.Target, replaceIndexRef)
		lbl.Target = AliasRefRegexOld.ReplaceAllStringFunc(lbl.Target, func(s string) string {
			logrus.Warnf("%q should now be %q, old syntax will be removed in a future version", lbl, strings.ReplaceAll(lbl.String(), "$", "#"))
			return replaceIndexRef(s)
		})
		if refErr.HasError() {
			b.Add(refErr.Build())
			return
		}
		config, ok := entries.Load(lbl.Target)
		if !ok {
			b.Add(E.NotExist("alias", lbl.Target))
			return
		}
		if err = D.ApplyLabel(config, lbl); err != nil {
			b.Add(err)
		}
	}
	return
}
