package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func setConfigField(pl *ProxyLabel, c *ProxyConfig) error {
	return setFieldFromSnake(c, pl.Field, pl.Value)
}

func (p *Provider) getContainerProxyConfigs(container *types.Container, clientIP string) (ProxyConfigSlice, error) {
	var aliases []string

	cfgs := make(ProxyConfigSlice, 0)
	cfgMap := make(map[string]*ProxyConfig)

	containerName := strings.TrimPrefix(container.Names[0], "/")
	aliasesLabel, ok := container.Labels["proxy.aliases"]

	if !ok {
		aliases = []string{containerName}
	} else {
		v, _ := commaSepParser(aliasesLabel)
		aliases = v.([]string)
	}

	if clientIP == "" && isHostNetworkMode {
		clientIP = "127.0.0.1"
	}
	isRemote := clientIP != ""

	for _, alias := range aliases {
		cfgMap[alias] = &ProxyConfig{}
	}

	ne := NewNestedError("these labels have errors").Subject(containerName)

	for label, value := range container.Labels {
		pl, err := parseProxyLabel(label, value)
		if err != nil {
			if !errors.Is(err, errNotProxyLabel) {
				ne.ExtraError(NewNestedErrorFrom(err).Subject(label))
			}
			continue
		}
		if pl.Alias == wildcardAlias {
			for alias := range cfgMap {
				pl.Alias = alias
				err = setConfigField(pl, cfgMap[alias])
				if err != nil {
					ne.ExtraError(NewNestedErrorFrom(err).Subject(pl.Alias))
				}
			}
			continue
		}
		config, ok := cfgMap[pl.Alias]
		if !ok {
			ne.ExtraError(NewNestedError("unknown alias").Subject(pl.Alias))
			continue
		}
		err = setConfigField(pl, config)
		if err != nil {
			ne.ExtraError(NewNestedErrorFrom(err).Subject(pl.Alias))
		}
	}

	for alias, config := range cfgMap {
		l := p.l.WithField("alias", alias)
		if config.Port == "" {
			config.Port = fmt.Sprintf("%d", selectPort(container, isRemote))
		}
		if config.Port == "0" {
			l.Infof("no ports exposed, ignored")
			continue
		}
		if config.Scheme == "" {
			switch {
			case strings.HasSuffix(config.Port, "443"):
				config.Scheme = "https"
			default:
				imageName := getImageName(container)
				_, isKnownImage := ImageNamePortMapTCP[imageName]
				if isKnownImage {
					config.Scheme = "tcp"
				} else {
					config.Scheme = "http"
				}
			}
		}
		if !isValidScheme(config.Scheme) {
			ne.Extra("unsupported scheme").Subject(config.Scheme)
		}

		if isRemote && strings.HasPrefix(config.Port, "*") {
			var err error
			// find matching port
			srcPort := config.Port[1:]
			config.Port, err = findMatchingContainerPort(container, srcPort)
			if err != nil {
				ne.ExtraError(NewNestedErrorFrom(err).Subjectf("alias %s", alias))
			}
			if isStreamScheme(config.Scheme) {
				config.Port = fmt.Sprintf("%s:%s", srcPort, config.Port)
			}
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

		if ne.HasExtras() {
			continue
		}
		cfgs = append(cfgs, *config)
	}

	if ne.HasExtras() {
		return nil, ne
	}
	return cfgs, nil
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

	ne := NewNestedError("these containers have errors")
	for _, container := range containerSlice {
		ccfgs, err := p.getContainerProxyConfigs(&container, clientIP)
		if err != nil {
			ne.ExtraError(err)
			continue
		}
		cfgs = append(cfgs, ccfgs...)
	}

	if ne.HasExtras() {
		// print but ignore
		p.l.Error(ne)
	}

	return cfgs, nil
}

// var dockerUrlRegex = regexp.MustCompile(`^(?P<scheme>\w+)://(?P<host>[^:]+)(?P<port>:\d+)?(?P<path>/.*)?$`)
func getImageName(c *types.Container) string {
	imageSplit := strings.Split(c.Image, "/")
	imageSplit = strings.Split(imageSplit[len(imageSplit)-1], ":")
	return imageSplit[0]
}

func getPublicPort(p types.Port) uint16  { return p.PublicPort }
func getPrivatePort(p types.Port) uint16 { return p.PrivatePort }

func selectPort(c *types.Container, isRemote bool) uint16 {
	if isRemote || c.HostConfig.NetworkMode == "host" {
		return selectPortInternal(c, getPublicPort)
	}
	return selectPortInternal(c, getPrivatePort)
}

// used when isRemote is true
func findMatchingContainerPort(c *types.Container, ps string) (string, error) {
	p, err := strconv.Atoi(ps)
	if err != nil {
		return "", err
	}
	pWant := uint16(p)
	for _, pGot := range c.Ports {
		if pGot.PrivatePort == pWant {
			return fmt.Sprintf("%d", pGot.PublicPort), nil
		}
	}
	return "", fmt.Errorf("port %d not found", p)
}

func selectPortInternal(c *types.Container, getPort func(types.Port) uint16) uint16 {
	imageName := getImageName(c)
	// if is known image -> use known port
	if port, isKnown := ImageNamePortMapHTTP[imageName]; isKnown {
		for _, p := range c.Ports {
			if p.PrivatePort == port {
				return getPort(p)
			}
		}
	}
	// if it has known http port -> use it
	for _, p := range c.Ports {
		if isWellKnownHTTPPort(p.PrivatePort) {
			return getPort(p)
		}
	}
	// if it has any port -> use it
	for _, p := range c.Ports {
		if port := getPort(p); port != 0 {
			return port
		}
	}
	return 0
}

func isWellKnownHTTPPort(port uint16) bool {
	_, ok := wellKnownHTTPPorts[port]
	return ok
}
