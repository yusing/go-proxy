package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func (p *Provider) setConfigField(c *ProxyConfig, label string, value string, prefix string) error {
	if strings.HasPrefix(label, prefix) {
		field := strings.TrimPrefix(label, prefix)
		if err := setFieldFromSnake(c, field, value); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) getContainerProxyConfigs(container types.Container, clientIP string) ProxyConfigSlice {
	var aliases []string

	cfgs := make(ProxyConfigSlice, 0)

	containerName := strings.TrimPrefix(container.Names[0], "/")
	aliasesLabel, ok := container.Labels["proxy.aliases"]

	if !ok {
		aliases = []string{containerName}
	} else {
		aliases = strings.Split(aliasesLabel, ",")
	}

	isRemote := clientIP != ""

	ne := NewNestedError("invalid label config").Subjectf("container %s", containerName)
	defer func() {
		if ne.HasExtras() {
			p.l.Error(ne)
		}
	}()

	for _, alias := range aliases {
		l := p.l.WithField("container", containerName).WithField("alias", alias)
		config := NewProxyConfig(p)
		prefix := fmt.Sprintf("proxy.%s.", alias)
		for label, value := range container.Labels {
			err := p.setConfigField(&config, label, value, prefix)
			if err != nil {
				ne.ExtraError(NewNestedErrorFrom(err).Subjectf("alias %s", alias))
			}
			err = p.setConfigField(&config, label, value, wildcardLabelPrefix)
			if err != nil {
				ne.ExtraError(NewNestedErrorFrom(err).Subjectf("alias %s", alias))
			}
		}
		if config.Port == "" {
			config.Port = fmt.Sprintf("%d", selectPort(&container))
		}
		if config.Port == "0" {
			l.Infof("no ports exposed, ignored")
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
			config.Host = containerName
		}
		config.Alias = alias

		cfgs = append(cfgs, config)
	}
	return cfgs
}

func (p *Provider) getDockerClient() (*client.Client, error) {
	var dockerOpts []client.Opt
	if p.Value == clientUrlFromEnv {
		dockerOpts = []client.Opt{
			client.WithHostFromEnv(),
			client.WithAPIVersionNegotiation(),
		}
	} else {
		helper, err := connhelper.GetConnectionHelper(p.Value)
		if err != nil {
			p.l.Fatal("unexpected error: ", err)
		}
		if helper != nil {
			httpClient := &http.Client{
				Transport: &http.Transport{
					DialContext: helper.Dialer,
				},
			}
			dockerOpts = []client.Opt{
				client.WithHTTPClient(httpClient),
				client.WithHost(helper.Host),
				client.WithAPIVersionNegotiation(),
				client.WithDialContext(helper.Dialer),
			}
		} else {
			dockerOpts = []client.Opt{
				client.WithHost(p.Value),
				client.WithAPIVersionNegotiation(),
			}
		}
	}
	return client.NewClientWithOpts(dockerOpts...)
}

func (p *Provider) getDockerProxyConfigs() (ProxyConfigSlice, error) {
	var clientIP string

	if p.Value == clientUrlFromEnv {
		clientIP = ""
	} else {
		url, err := client.ParseHostURL(p.Value)
		if err != nil {
			return nil, NewNestedError("invalid host url").Subject(p.Value).With(err)
		}
		clientIP = strings.Split(url.Host, ":")[0]
	}

	dockerClient, err := p.getDockerClient()

	if err != nil {
		return nil, NewNestedError("unable to create docker client").With(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	containerSlice, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true})

	if err != nil {
		return nil, NewNestedError("unable to list containers").With(err)
	}

	cfgs := make(ProxyConfigSlice, 0)

	for _, container := range containerSlice {
		cfgs = append(cfgs, p.getContainerProxyConfigs(container, clientIP)...)
	}

	return cfgs, nil
}

// var dockerUrlRegex = regexp.MustCompile(`^(?P<scheme>\w+)://(?P<host>[^:]+)(?P<port>:\d+)?(?P<path>/.*)?$`)

func getPublicPort(p types.Port) uint16  { return p.PublicPort }
func getPrivatePort(p types.Port) uint16 { return p.PrivatePort }

func selectPort(c *types.Container) uint16 {
	if c.HostConfig.NetworkMode == "host" {
		return selectPortInternal(c, getPublicPort)
	}
	return selectPortInternal(c, getPrivatePort)
}

func selectPortInternal(c *types.Container, getPort func(types.Port) uint16) uint16 {
	for _, p := range c.Ports {
		if port := getPort(p); port != 0 {
			return port
		}
	}
	return 0
}
