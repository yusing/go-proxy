package main

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ProxyConfig struct {
	id     string
	Alias  string
	Scheme string
	Host   string
	Port   string
	Path   string // http proxy only
}

func NewProxyConfig() ProxyConfig {
	return ProxyConfig{}
}

func (cfg *ProxyConfig) UpdateId() {
	cfg.id = fmt.Sprintf("%s-%s-%s-%s-%s", cfg.Alias, cfg.Scheme, cfg.Host, cfg.Port, cfg.Path)
}

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
				field = cases.Title(language.Und, cases.NoLower).String(field)
				prop := reflect.ValueOf(&config).Elem().FieldByName(field)
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
				_, isKnownImage := imageNamePortMap[imageName]
				if isKnownImage {
					config.Scheme = "tcp"
				} else {
					config.Scheme = "http"
				}
			}
		}
		if !isValidScheme(config.Scheme) {
			log.Printf("%s: unsupported scheme: %s, using http", container_name, config.Scheme)
			config.Scheme = "http"
		}
		if config.Host == "" {
			if container.HostConfig.NetworkMode != "host" {
				config.Host = container_name
			} else {
				config.Host = "host.docker.internal"
			}
		}
		config.Alias = alias
		config.UpdateId()

		wg.Add(1)
		go func() {
			createRoute(&config)
			wg.Done()
		}()
	}
	wg.Wait()
}

func buildRoutes() {
	initRoutes()
	containerSlice, err := dockerClient.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "go-proxy"
	}
	for _, container := range containerSlice {
		if container.Names[0] == hostname { // skip self
			continue
		}
		buildContainerRoute(container)
	}
}

func findHTTPRoute(host string, path string) (*HTTPRoute, error) {
	subdomain := strings.Split(host, ".")[0]
	routeMap, ok := routes.HTTPRoutes.TryGet(subdomain)
	if !ok {
		return nil, fmt.Errorf("no matching route for subdomain %s", subdomain)
	}
	for _, route := range routeMap {
		if strings.HasPrefix(path, route.Path) {
			return &route, nil
		}
	}
	return nil, fmt.Errorf("no matching route for path %s for subdomain %s", path, subdomain)
}
