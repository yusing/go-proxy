package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func (p *Provider) setConfigField(c *ProxyConfig, label string, value string, prefix string) error {
	if strings.HasPrefix(label, prefix) {
		field := strings.TrimPrefix(label, prefix)
		field = utils.snakeToCamel(field)
		prop := reflect.ValueOf(c).Elem().FieldByName(field)
		if prop.Kind() == 0 {
			return fmt.Errorf("ignoring unknown field %s", field)
		}
		prop.Set(reflect.ValueOf(value))
	}
	return nil
}

func (p *Provider) getContainerProxyConfigs(container types.Container, clientIP string) []*ProxyConfig {
	var aliases []string

	cfgs := make([]*ProxyConfig, 0)

	container_name := strings.TrimPrefix(container.Names[0], "/")
	aliases_label, ok := container.Labels["proxy.aliases"]

	if !ok {
		aliases = []string{container_name}
	} else {
		aliases = strings.Split(aliases_label, ",")
	}

	isRemote := clientIP != ""

	for _, alias := range aliases {
		l := p.l.WithField("container", container_name).WithField("alias", alias)
		config := NewProxyConfig(p)
		prefix := fmt.Sprintf("proxy.%s.", alias)
		for label, value := range container.Labels {
			err := p.setConfigField(&config, label, value, prefix)
			if err != nil {
				l.Error(err)
			}
			err = p.setConfigField(&config, label, value, wildcardPrefix)
			if err != nil {
				l.Error(err)
			}
		}
		if config.Port == "" {
			config.Port = fmt.Sprintf("%d", selectPort(container))
		}
		if config.Port == "0" {
			// no ports exposed or specified
			l.Info("no ports exposed, ignored")
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
			l.Warnf("unsupported scheme: %s, using http", config.Scheme)
			config.Scheme = "http"
		}
		if config.Host == "" {
			switch {
			case isRemote:
				config.Host = clientIP
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
	var clientIP string
	var opts []client.Opt
	var err error

	if p.Value == clientUrlFromEnv {
		clientIP = ""
		opts = []client.Opt{
			client.WithHostFromEnv(),
			client.WithAPIVersionNegotiation(),
		}
	} else {
		url, err := client.ParseHostURL(p.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to parse docker host url: %v", err)
		}
		clientIP = strings.Split(url.Host, ":")[0]
		helper, err := connhelper.GetConnectionHelper(p.Value)
		if err != nil {
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
		if helper != nil {
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
		} else {
			opts = []client.Opt{
				client.WithHost(p.Value),
				client.WithAPIVersionNegotiation(),
			}
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
		cfgs = append(cfgs, p.getContainerProxyConfigs(container, clientIP)...)
	}

	return cfgs, nil
}

// var dockerUrlRegex = regexp.MustCompile(`^(?P<scheme>\w+)://(?P<host>[^:]+)(?P<port>:\d+)?(?P<path>/.*)?$`)

func getPublicPort(p types.Port) uint16  { return p.PublicPort }
func getPrivatePort(p types.Port) uint16 { return p.PrivatePort }

func selectPort(c types.Container) uint16 {
	if c.HostConfig.NetworkMode == "host" {
		return selectPortInternal(c, getPrivatePort)
	}
	return selectPortInternal(c, getPublicPort)
}

func selectPortInternal(c types.Container, getPort func(types.Port) uint16) uint16 {
	for _, p := range c.Ports {
		if port := getPort(p); port != 0 {
			return port
		}
	}
	return 0
}

const wildcardPrefix = "proxy.*."
