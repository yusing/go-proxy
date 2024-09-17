package provider

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	PT "github.com/yusing/go-proxy/proxy/fields"
	W "github.com/yusing/go-proxy/watcher"
)

type DockerProvider struct {
	dockerHost string
}

func DockerProviderImpl(dockerHost string) ProviderImpl {
	return &DockerProvider{dockerHost: dockerHost}
}

// GetProxyEntries returns proxy entries from a docker client.
//
// It retrieves the docker client information using the dockerhelper.GetClientInfo method.
// Then, it iterates over the containers in the docker client information and calls
// the getEntriesFromLabels method to get the proxy entries for each container.
// Any errors encountered during the process are added to the ne error object.
// Finally, it returns the collected proxy entries and the ne error object.
//
// Parameters:
//   - p: A pointer to the DockerProvider struct.
//
// Returns:
//   - P.EntryModelSlice: (non-nil) A slice of EntryModel structs representing the proxy entries.
//   - error: An error object if there was an error retrieving the docker client information or parsing the labels.
func (p DockerProvider) GetProxyEntries() (M.ProxyEntries, E.NestedError) {
	entries := M.NewProxyEntries()

	info, err := D.GetClientInfo(p.dockerHost)
	if err.HasError() {
		return entries, err
	}

	errors := E.NewBuilder("errors when parse docker labels")

	for _, container := range info.Containers {
		en, err := p.getEntriesFromLabels(&container, info.Host)
		if err.HasError() {
			errors.Add(err)
		}
		// although err is not nil
		// there may be some valid entries in `en`
		dups := entries.MergeWith(en)
		// add the duplicate proxy entries to the error
		dups.EachKV(func(k string, v *M.ProxyEntry) {
			errors.Addf("duplicate alias %s", k)
		})
	}

	return entries, errors.Build()
}

func (p *DockerProvider) NewWatcher() W.Watcher {
	return W.NewDockerWatcher(p.dockerHost)
}

// Returns a list of proxy entries for a container.
// Always non-nil
func (p *DockerProvider) getEntriesFromLabels(container *types.Container, clientHost string) (M.ProxyEntries, E.NestedError) {
	var mainAlias string
	var aliases PT.Aliases

	// set mainAlias to docker compose service name if available
	if serviceName, ok := container.Labels["com.docker.compose.service"]; ok {
		mainAlias = serviceName
	}

	// if mainAlias is not set,
	// or container name is different from service name
	// use container name
	if containerName := strings.TrimPrefix(container.Names[0], "/"); containerName != mainAlias {
		mainAlias = containerName
	}

	if l, ok := container.Labels["proxy.aliases"]; ok {
		aliases = PT.NewAliases(l)
		delete(container.Labels, "proxy.aliases")
	} else {
		aliases = PT.NewAliases(mainAlias)
	}

	entries := M.NewProxyEntries()

	// find first port, return if no port exposed
	defaultPort, err := findFirstPort(container)
	if err.HasError() {
		logrus.Debug(mainAlias, " ", err.Error())
	}

	// init entries map for all aliases
	aliases.ForEach(func(a PT.Alias) {
		entries.Set(string(a), &M.ProxyEntry{
			Alias: string(a),
			Host:  clientHost,
			Port:  defaultPort,
		})
	})

	errors := E.NewBuilder("failed to apply label for %q", mainAlias)
	for key, val := range container.Labels {
		lbl, err := D.ParseLabel(key, val)
		if err.HasError() {
			errors.Add(E.From(err).Subject(key))
			continue
		}
		if lbl.Namespace != D.NSProxy {
			continue
		}
		if lbl.Target == wildcardAlias {
			// apply label for all aliases
			entries.EachKV(func(a string, e *M.ProxyEntry) {
				if err = D.ApplyLabel(e, lbl); err.HasError() {
					errors.Add(E.From(err).Subject(lbl.Target))
				}
			})
		} else {
			config, ok := entries.UnsafeGet(lbl.Target)
			if !ok {
				errors.Add(E.NotExists("alias", lbl.Target))
				continue
			}
			if err = D.ApplyLabel(config, lbl); err.HasError() {
				errors.Add(err.Subject(lbl.Target))
			}
		}
	}

	entries.EachKV(func(a string, e *M.ProxyEntry) {
		if e.Port == "" {
			entries.UnsafeDelete(a)
		}
	})

	return entries, errors.Build()
}

func findFirstPort(c *types.Container) (string, E.NestedError) {
	if len(c.Ports) == 0 {
		return "", E.FailureWhy("findFirstPort", "no port exposed")
	}
	for _, p := range c.Ports {
		if p.PublicPort != 0 {
			return fmt.Sprint(p.PublicPort), E.Nil()
		}
	}
	return "", E.Failure("findFirstPort")
}
