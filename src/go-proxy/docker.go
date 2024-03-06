package main

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

var dockerClient *client.Client

func buildContainerRoute(container types.Container) {
	var aliases []string
	var wg sync.WaitGroup

	container_name := strings.TrimPrefix(container.Names[0], "/")
	aliases_label, ok := container.Labels["proxy.aliases"]
	if !ok {
		aliases = []string{container_name}
	} else {
		aliases = strings.Split(aliases_label, ",")
	}

	for _, alias := range aliases {
		config := NewProxyConfig()
		prefix := fmt.Sprintf("proxy.%s.", alias)
		for label, value := range container.Labels {
			if strings.HasPrefix(label, prefix) {
				field := strings.TrimPrefix(label, prefix)
				field = utils.snakeToCamel(field)
				prop := reflect.ValueOf(&config).Elem().FieldByName(field)
				if prop.Kind() == 0 {
					glog.Infof("[Build] %s: ignoring unknown field %s", alias, field)
					continue
				}
				prop.Set(reflect.ValueOf(value))
			}
		}
		if config.Port == "" {
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
			glog.Infof("[Build] %s has no port exposed", alias)
			return
		}
		if config.Scheme == "" {
			if strings.HasSuffix(config.Port, "443") {
				config.Scheme = "https"
			} else if strings.HasPrefix(container.Image, "sha256:") {
				config.Scheme = "http"
			} else {
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
			glog.Infof("%s: unsupported scheme: %s, using http", container_name, config.Scheme)
			config.Scheme = "http"
		}
		if config.Host == "" {
			switch {
			case container.HostConfig.NetworkMode == "host":
				config.Host = "host.docker.internal"
			case config.LoadBalance == "true":
			case config.LoadBalance == "1":
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
		config.UpdateId()

		wg.Add(1)
		go func() {
			CreateRoute(&config)
			wg.Done()
		}()
	}
	wg.Wait()
}

func buildRoutes() {
	InitRoutes()
	containerSlice, err := dockerClient.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		glog.Fatal(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "go-proxy"
	}
	for _, container := range containerSlice {
		if container.Names[0] == hostname { // skip self
			glog.Infof("[Build] Skipping %s", container.Names[0])
			continue
		}
		buildContainerRoute(container)
	}
}
