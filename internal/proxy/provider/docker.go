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
	R "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/types"
	W "github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type DockerProvider struct {
	name, dockerHost string
	ExplicitOnly     bool
}

var (
	AliasRefRegex    = regexp.MustCompile(`#\d+`)
	AliasRefRegexOld = regexp.MustCompile(`\$\d+`)
)

func DockerProviderImpl(name, dockerHost string, explicitOnly bool) (ProviderImpl, E.NestedError) {
	if dockerHost == common.DockerHostFromEnv {
		dockerHost = common.GetEnv("DOCKER_HOST", client.DefaultDockerHost)
	}
	return &DockerProvider{name, dockerHost, explicitOnly}, nil
}

func (p *DockerProvider) String() string {
	return "docker: " + p.name
}

func (p *DockerProvider) NewWatcher() W.Watcher {
	return W.NewDockerWatcher(p.dockerHost)
}

func (p *DockerProvider) LoadRoutesImpl() (routes R.Routes, err E.NestedError) {
	routes = R.NewRoutes()
	entries := types.NewProxyEntries()

	info, err := D.GetClientInfo(p.dockerHost, true)
	if err != nil {
		return routes, E.FailWith("connect to docker", err)
	}

	errors := E.NewBuilder("errors in docker labels")

	for _, c := range info.Containers {
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
		dups.RangeAll(func(k string, v *types.RawEntry) {
			errors.Addf("duplicate alias %s", k)
		})
	}

	entries.RangeAll(func(_ string, e *types.RawEntry) {
		e.DockerHost = p.dockerHost
	})

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

func (p *DockerProvider) OnEvent(event W.Event, routes R.Routes) (res EventResult) {
	switch event.Action {
	case events.ActionContainerStart, events.ActionContainerStop:
		break
	default:
		return
	}
	b := E.NewBuilder("event %s error", event)
	defer b.To(&res.err)

	routes.RangeAll(func(k string, v *R.Route) {
		if v.Entry.ContainerID == event.ActorID ||
			v.Entry.ContainerName == event.ActorName {
			b.Add(v.Stop())
			routes.Delete(k)
			res.nRemoved++
		}
	})

	if res.nRemoved == 0 { // id & container name changed
		// load all routes (rescan)
		routesNew, err := p.LoadRoutesImpl()
		routesOld := routes
		if routesNew.Size() == 0 {
			b.Add(E.FailWith("rescan routes", err))
			return
		}
		routesNew.Range(func(k string, v *R.Route) bool {
			if !routesOld.Has(k) {
				routesOld.Store(k, v)
				b.Add(v.Start())
				res.nAdded++
				return false
			}
			return true
		})
		routesOld.Range(func(k string, v *R.Route) bool {
			if !routesNew.Has(k) {
				b.Add(v.Stop())
				routesOld.Delete(k)
				res.nRemoved++
				return false
			}
			return true
		})
		return
	}

	client, err := D.ConnectClient(p.dockerHost)
	if err != nil {
		b.Add(E.FailWith("connect to docker", err))
		return
	}
	defer client.Close()
	cont, err := client.Inspect(event.ActorID)
	if err != nil {
		b.Add(E.FailWith("inspect container", err))
		return
	}

	if p.shouldIgnore(cont) {
		return
	}

	entries, err := p.entriesFromContainerLabels(cont)
	b.Add(err)

	entries.RangeAll(func(alias string, entry *types.RawEntry) {
		if routes.Has(alias) {
			b.Add(E.Duplicated("alias", alias))
		} else {
			if route, err := R.NewRoute(entry); err != nil {
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
// Always non-nil.
func (p *DockerProvider) entriesFromContainerLabels(container *D.Container) (entries types.RawEntries, _ E.NestedError) {
	entries = types.NewProxyEntries()

	if p.shouldIgnore(container) {
		return
	}

	// init entries map for all aliases
	for _, a := range container.Aliases {
		entries.Store(a, &types.RawEntry{
			Alias:     a,
			Container: container,
		})
	}

	errors := E.NewBuilder("failed to apply label")
	for key, val := range container.Labels {
		errors.Add(p.applyLabel(container, entries, key, val))
	}

	// remove all entries that failed to fill in missing fields
	entries.RangeAll(func(_ string, re *types.RawEntry) {
		re.FillMissingFields()
	})

	return entries, errors.Build().Subject(container.ContainerName)
}

func (p *DockerProvider) applyLabel(container *D.Container, entries types.RawEntries, key, val string) (res E.NestedError) {
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
		entries.RangeAll(func(a string, e *types.RawEntry) {
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
