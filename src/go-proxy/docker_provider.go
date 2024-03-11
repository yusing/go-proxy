package main

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func (p *Provider) getContainerProxyConfigs(container types.Container, clientHost string) []*ProxyConfig {
	var aliases []string

	cfgs := make([]*ProxyConfig, 0)

	container_name := strings.TrimPrefix(container.Names[0], "/")
	aliases_label, ok := container.Labels["proxy.aliases"]
	if !ok {
		aliases = []string{container_name}
	} else {
		aliases = strings.Split(aliases_label, ",")
	}

	for _, alias := range aliases {
		config := NewProxyConfig(p)
		prefix := fmt.Sprintf("proxy.%s.", alias)
		for label, value := range container.Labels {
			if strings.HasPrefix(label, prefix) {
				field := strings.TrimPrefix(label, prefix)
				field = utils.snakeToCamel(field)
				prop := reflect.ValueOf(&config).Elem().FieldByName(field)
				if prop.Kind() == 0 {
					p.Logf("Build", "ignoring unknown field %s", alias, field)
					continue
				}
				prop.Set(reflect.ValueOf(value))
			}
		}
		if config.Port == "" && clientHost != "" {
			for _, port := range container.Ports {
				config.Port = fmt.Sprintf("%d", port.PublicPort)
				break
			}
		} else if config.Port == "" {
			// usually the smaller port is the http one
			// so make it the last one to be set (if 80 or 8080 are not exposed)
			sort.Slice(container.Ports, func(i, j int) bool {
				return container.Ports[i].PrivatePort > container.Ports[j].PrivatePort
			})
			for _, port := range container.Ports {
				// set first, but keep trying
				config.Port = fmt.Sprintf("%d", port.PrivatePort)
				// until we find 80 or 8080
				if port.PrivatePort == 80 || port.PrivatePort == 8080 {
					break
				}
			}
		}
		if config.Port == "" {
			// no ports exposed or specified
			p.Logf("Build", "no ports exposed for %s, ignored", container_name)
			continue
		}
		if config.Scheme == "" {
			switch {
			case strings.HasSuffix(config.Port, "443"):
				config.Scheme = "https"
			case strings.HasPrefix(container.Image, "sha256:"):
				config.Scheme = "http"
			default:
				imageSplit := strings.Split(container.Image, "/")
				imageSplit = strings.Split(imageSplit[len(imageSplit)-1], ":")
				imageName := imageSplit[0]
				_, isKnownImage := ImageNamePortMap[imageName]
				if isKnownImage {
					config.Scheme = "tcp"
				} else {
					config.Scheme = "http"
				}
			}
		}
		if !isValidScheme(config.Scheme) {
			p.Warningf("Build", "unsupported scheme: %s, using http", container_name, config.Scheme)
			config.Scheme = "http"
		}
		if config.Host == "" {
			switch {
			case clientHost != "":
				config.Host = clientHost
			case container.HostConfig.NetworkMode == "host":
				config.Host = "host.docker.internal"
			case config.LoadBalance == "true", config.LoadBalance == "1":
				for _, network := range container.NetworkSettings.Networks {
					config.Host = network.IPAddress
					break
				}
			default:
				for _, network := range container.NetworkSettings.Networks {
					for _, alias := range network.Aliases {
						config.Host = alias
						break
					}
				}
			}
		}
		if config.Host == "" {
			config.Host = container_name
		}
		config.Alias = alias

		cfgs = append(cfgs, &config)
	}
	return cfgs
}

func (p *Provider) getDockerProxyConfigs() ([]*ProxyConfig, error) {
	var clientHost string
	var opts []client.Opt
	var err error

	if p.Value == clientUrlFromEnv {
		clientHost = ""
		opts = []client.Opt{
			client.WithHostFromEnv(),
			client.WithAPIVersionNegotiation(),
		}
	} else {
		url, err := client.ParseHostURL(p.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to parse docker host url: %v", err)
		}
		clientHost = url.Host
		helper, err := connhelper.GetConnectionHelper(p.Value)
		if err != nil {
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
		httpClient := &http.Client{
			Transport: &http.Transport{
				DialContext: helper.Dialer,
			},
		}
		opts = []client.Opt{
			client.WithHTTPClient(httpClient),
			client.WithHost(helper.Host),
			client.WithAPIVersionNegotiation(),
			client.WithDialContext(helper.Dialer),
		}
	}

	p.dockerClient, err = client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %v", err)
	}

	containerSlice, err := p.dockerClient.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %v", err)
	}

	cfgs := make([]*ProxyConfig, 0)

	for _, container := range containerSlice {
		cfgs = append(cfgs, p.getContainerProxyConfigs(container, clientHost)...)
	}

	return cfgs, nil
}

func (p *Provider) grWatchDockerChanges() {
	p.stopWatching = make(chan struct{})

	filter := filters.NewArgs(
		filters.Arg("type", "container"),
		filters.Arg("event", "start"),
		filters.Arg("event", "die"), // 'stop' already triggering 'die'
	)
	msgChan, errChan := p.dockerClient.Events(context.Background(), types.EventsOptions{Filters: filter})

	for {
		select {
		case <-p.stopWatching:
			return
		case msg := <-msgChan:
			// TODO: handle actor only
			p.Logf("Event", "%s %s caused rebuild", msg.Action, msg.Actor.Attributes["name"])
			p.StopAllRoutes()
			p.BuildStartRoutes()
		case err := <-errChan:
			p.Logf("Event", "error %s", err)
			msgChan, errChan = p.dockerClient.Events(context.Background(), types.EventsOptions{Filters: filter})
		}
	}
}

// var dockerUrlRegex = regexp.MustCompile(`^(?P<scheme>\w+)://(?P<host>[^:]+)(?P<port>:\d+)?(?P<path>/.*)?$`)
