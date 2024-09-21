package provider

import (
	"strconv"

	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	R "github.com/yusing/go-proxy/route"
	W "github.com/yusing/go-proxy/watcher"
)

type DockerProvider struct {
	dockerHost, hostname string
}

func DockerProviderImpl(dockerHost string) ProviderImpl {
	return &DockerProvider{dockerHost: dockerHost}
}

func (p *DockerProvider) NewWatcher() W.Watcher {
	return W.NewDockerWatcher(p.dockerHost)
}

func (p *DockerProvider) LoadRoutesImpl() (routes R.Routes, err E.NestedError) {
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
		dups.RangeAll(func(k string, v *M.ProxyEntry) {
			errors.Addf("duplicate alias %s", k)
		})
	}

	entries.RangeAll(func(_ string, e *M.ProxyEntry) {
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

	entries.RangeAll(func(alias string, entry *M.ProxyEntry) {
		if routes.Has(alias) {
			b.Add(E.AlreadyExist("alias", alias))
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
func (p *DockerProvider) entriesFromContainerLabels(container D.Container) (M.ProxyEntries, E.NestedError) {
	entries := M.NewProxyEntries()

	// init entries map for all aliases
	for _, a := range container.Aliases {
		entries.Store(a, &M.ProxyEntry{
			Alias:           a,
			Host:            p.hostname,
			ProxyProperties: container.ProxyProperties,
		})
	}

	errors := E.NewBuilder("failed to apply label")
	for key, val := range container.Labels {
		errors.Add(p.applyLabel(entries, key, val))
	}

	// selecting correct host port
	if container.HostConfig.NetworkMode != "host" {
		for _, a := range container.Aliases {
			entry, ok := entries.Load(a)
			if !ok {
				continue
			}
			for _, p := range container.Ports {
				containerPort := strconv.Itoa(int(p.PrivatePort))
				if containerPort == entry.Port {
					entry.Port = strconv.Itoa(int(p.PublicPort))
				}
			}
		}
	}

	return entries, errors.Build().Subject(container.ContainerName)
}

func (p *DockerProvider) applyLabel(entries M.ProxyEntries, key, val string) (res E.NestedError) {
	b := E.NewBuilder("errors in label %s", key)
	defer b.To(&res)

	lbl, err := D.ParseLabel(key, val)
	if err.HasError() {
		b.Add(err.Subject(key))
	}
	if lbl.Namespace != D.NSProxy {
		return
	}
	if lbl.Target == D.WildcardAlias {
		// apply label for all aliases
		entries.RangeAll(func(a string, e *M.ProxyEntry) {
			if err = D.ApplyLabel(e, lbl); err.HasError() {
				b.Add(err.Subject(lbl.Target))
			}
		})
	} else {
		config, ok := entries.Load(lbl.Target)
		if !ok {
			b.Add(E.NotExist("alias", lbl.Target))
			return
		}
		if err = D.ApplyLabel(config, lbl); err.HasError() {
			b.Add(err.Subject(lbl.Target))
		}
	}
	return
}
