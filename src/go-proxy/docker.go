package go_proxy

import (
	"fmt"
	"log"
	"net/url"
	"reflect"
	"sort"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Config struct {
	Scheme string
	Host   string
	Port   string
	Path   string
}

type Route struct {
	Url  *url.URL
	Path string
}

var dockerClient *client.Client
var subdomainRouteMap map[string]mapset.Set[Route] // subdomain -> path

func buildContainerCfg(container types.Container) {
	var aliases []string

	container_name := strings.TrimPrefix(container.Names[0], "/")
	aliases_label, ok := container.Labels["proxy.aliases"]
	if !ok {
		aliases = []string{container_name}
	} else {
		aliases = strings.Split(aliases_label, ",")
	}

	for _, alias := range aliases {
		config := NewConfig()
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
			} else {
				config.Scheme = "http"
			}
		}
		if config.Scheme != "http" && config.Scheme != "https" {
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
		_, inMap := subdomainRouteMap[alias]
		if !inMap {
			subdomainRouteMap[alias] = mapset.NewSet[Route]()
		}
		url, err := url.Parse(fmt.Sprintf("%s://%s:%s", config.Scheme, config.Host, config.Port))
		if err != nil {
			log.Fatal(err)
		}
		subdomainRouteMap[alias].Add(Route{Url: url, Path: config.Path})
	}
}

func buildRoutes() {
	subdomainRouteMap = make(map[string]mapset.Set[Route])
	containerSlice, err := dockerClient.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, container := range containerSlice {
		buildContainerCfg(container)
	}
	subdomainRouteMap["go-proxy"] = panelRoute
}
