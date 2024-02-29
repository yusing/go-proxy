package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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
	Url  url.URL
	Path string
}

var dockerClient *client.Client
var subdomainRouteMap map[string][]Route // subdomain -> path

func NewConfig() Config {
	return Config{Scheme: "http", Host: "", Port: "", Path: ""}
}

func main() {
	var err error
	runtime.GOMAXPROCS(runtime.NumCPU())

	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		filter := filters.NewArgs(
			filters.Arg("type", "container"),
			filters.Arg("event", "start"),
			filters.Arg("event", "die"), // stop seems like triggering die
			// filters.Arg("event", "stop"),
		)
		msgs, _ := dockerClient.Events(context.Background(), types.EventsOptions{Filters: filter})

		for msg := range msgs {
			// TODO: handle actor only
			log.Printf("[Event] %s %s caused rebuild", msg.Action, msg.Actor.Attributes["name"])
			buildRoutes()
			log.Printf("[Build] rebuilt %v reverse proxies", len(subdomainRouteMap))
		}
	}()

	buildRoutes()
	log.Printf("[Build] built %v reverse proxies", len(subdomainRouteMap))

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	http.HandleFunc("/", handler)
	go func() {
		log.Println("Starting HTTP server on port 80")
		err := http.ListenAndServe(":80", nil)
		if err != nil {
			log.Fatal("HTTP server error", err)
		}
	}()
	log.Println("Starting HTTPS server on port 443")
	err = http.ListenAndServeTLS(":443", "/certs/cert.crt", "/certs/priv.key", nil)
	if err != nil {
		log.Fatal("HTTPS Server error: ", err)
	}
}

func redirectTLS(w http.ResponseWriter, r *http.Request) {
	// Redirect to the same host but with HTTPS
	http.Redirect(w, r, "https://"+r.Host+r.URL.Path, http.StatusMovedPermanently)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil {
		redirectTLS(w, r)
		return
	}
	subdomain := strings.Split(r.Host, ".")[0]
	// log.Printf("[Request] %s%s\n", r.Host, r.URL)

	routeMap, ok := subdomainRouteMap[subdomain]
	if !ok {
		http.Error(w, fmt.Sprintf("no matching route for subdomain %s", subdomain), http.StatusNotFound)
		return
	}
	for _, route := range routeMap {
		if strings.HasPrefix(r.URL.Path, route.Path) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, route.Path)
			// log.Printf("[Route] %s", route.Url.String())

			proxy := httputil.NewSingleHostReverseProxy(&route.Url)
			proxy.ServeHTTP(w, r)
			return
		}
	}
	http.Error(w, fmt.Sprintf("no matching route for path %s for subdomain %s", r.URL.Path, subdomain), http.StatusNotFound)
}

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
		if config.Scheme != "http" && config.Scheme != "https" {
			log.Printf("%s: unsupported scheme: %s, using http", container_name, config.Scheme)
			config.Scheme = "http"
		}
		if config.Port == "" {
			for _, port := range container.Ports {
				config.Port = fmt.Sprintf("%d", port.PrivatePort)
				break
			}
		}
		if config.Port == "" {
			// no ports exposed or specified
			return
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
			subdomainRouteMap[alias] = make([]Route, 0)
		}
		route := Route{Url: url.URL{Scheme: config.Scheme, Host: fmt.Sprintf("%s:%s", config.Host, config.Port)}, Path: config.Path}
		subdomainRouteMap[alias] = append(subdomainRouteMap[alias], route)
	}
}
func buildRoutes() {
	subdomainRouteMap = make(map[string][]Route)
	containerSlice, err := dockerClient.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, container := range containerSlice {
		buildContainerCfg(container)
	}
	// log.Println(subdomainRouteMap)
}
