package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	M "github.com/yusing/go-proxy/internal/models"
	R "github.com/yusing/go-proxy/internal/route"
	W "github.com/yusing/go-proxy/internal/watcher"
)

type DockerProvider struct {
	dockerHost, hostname string
	ExplicitOnly         bool
}

var AliasRefRegex = regexp.MustCompile(`#\d+`)
var AliasRefRegexOld = regexp.MustCompile(`\$\d+`)

func DockerProviderImpl(dockerHost string, explicitOnly bool) (ProviderImpl, E.NestedError) {
	hostname, err := D.ParseDockerHostname(dockerHost)
	if err.HasError() {
		return nil, err
	}
	return &DockerProvider{dockerHost, hostname, explicitOnly}, nil
}

func (p *DockerProvider) String() string {
	return fmt.Sprintf("docker:%s", p.dockerHost)
}

func (p *DockerProvider) NewWatcher() W.Watcher {
	return W.NewDockerWatcher(p.dockerHost)
}

func (p *DockerProvider) LoadRoutesImpl() (routes R.Routes, err E.NestedError) {
	routes = R.NewRoutes()
	entries := M.NewProxyEntries()

	info, err := D.GetClientInfo(p.dockerHost, true)
	if err.HasError() {
		return routes, E.FailWith("connect to docker", err)
	}

	errors := E.NewBuilder("errors when parse docker labels")

	for _, c := range info.Containers {
		container := D.FromDocker(&c, p.dockerHost)
		if container.IsExcluded {
			continue
		}

		newEntries, err := p.entriesFromContainerLabels(container)
		if err.HasError() {
			errors.Add(err)
		}
		// although err is not nil
		// there may be some valid entries in `en`
		dups := entries.MergeFrom(newEntries)
		// add the duplicate proxy entries to the error
		dups.RangeAll(func(k string, v *M.RawEntry) {
			errors.Addf("duplicate alias %s", k)
		})
	}

	entries.RangeAll(func(_ string, e *M.RawEntry) {
		e.DockerHost = p.dockerHost
	})

	routes, err = R.FromEntries(entries)
	errors.Add(err)

	return routes, errors.Build()
}

func (p *DockerProvider) OnEvent(event W.Event, routes R.Routes) (res EventResult) {
	b := E.NewBuilder("event %s error", event)
	defer b.To(&res.err)

	routes.RangeAll(func(k string, v R.Route) {
		if v.Entry().ContainerName == event.ActorName {
			b.Add(v.Stop())
			routes.Delete(k)
			res.nRemoved++
		}
	})

	client, err := D.ConnectClient(p.dockerHost)
	if err.HasError() {
		b.Add(E.FailWith("connect to docker", err))
		return
	}
	defer client.Close()
	cont, err := client.Inspect(event.ActorID)
	if err.HasError() {
		b.Add(E.FailWith("inspect container", err))
		return
	}
	entries, err := p.entriesFromContainerLabels(cont)
	b.Add(err)

	entries.RangeAll(func(alias string, entry *M.RawEntry) {
		if routes.Has(alias) {
			b.Add(E.Duplicated("alias", alias))
		} else {
			if route, err := R.NewRoute(entry); err.HasError() {
				b.Add(err)
			} else {
				routes.Store(alias, route)
				b.Add(route.Start())
				res.nAdded++
			}
		}
	})

	return
}

// Returns a list of proxy entries for a container.
// Always non-nil
func (p *DockerProvider) entriesFromContainerLabels(container D.Container) (entries M.RawEntries, _ E.NestedError) {
	entries = M.NewProxyEntries()

	if container.IsExcluded ||
		!container.IsExplicit && p.ExplicitOnly {
		return
	}

	// init entries map for all aliases
	for _, a := range container.Aliases {
		entries.Store(a, &M.RawEntry{
			Alias:           a,
			Host:            p.hostname,
			ProxyProperties: container.ProxyProperties,
		})
	}

	errors := E.NewBuilder("failed to apply label")
	for key, val := range container.Labels {
		errors.Add(p.applyLabel(container, entries, key, val))
	}

	// remove all entries that failed to fill in missing fields
	entries.RemoveAll(func(re *M.RawEntry) bool {
		return !re.FillMissingFields()
	})

	return entries, errors.Build().Subject(container.ContainerName)
}

func (p *DockerProvider) applyLabel(container D.Container, entries M.RawEntries, key, val string) (res E.NestedError) {
	b := E.NewBuilder("errors in label %s", key)
	defer b.To(&res)

	refErr := E.NewBuilder("errors parsing alias references")
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
	if err.HasError() {
		b.Add(err.Subject(key))
	}
	if lbl.Namespace != D.NSProxy {
		return
	}
	if lbl.Target == D.WildcardAlias {
		// apply label for all aliases
		entries.RangeAll(func(a string, e *M.RawEntry) {
			if err = D.ApplyLabel(e, lbl); err.HasError() {
				b.Add(err.Subjectf("alias %s", lbl.Target))
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
		if err = D.ApplyLabel(config, lbl); err.HasError() {
			b.Add(err.Subjectf("alias %s", lbl.Target))
		}
	}
	return
}
