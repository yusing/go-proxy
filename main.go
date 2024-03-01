package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"time"

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

var transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          1000,
	MaxIdleConnsPerHost:   1000,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
}

func NewConfig() Config {
	return Config{Scheme: "", Host: "", Port: "", Path: ""}
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

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	go func() {
		log.Println("Starting HTTP server on port 80")
		err := http.ListenAndServe(":80", http.HandlerFunc(redirectToTLS))
		if err != nil {
			log.Fatal("HTTP server error", err)
		}
	}()
	log.Println("Starting HTTPS server on port 443")
	err = http.ListenAndServeTLS(":443", "/certs/cert.crt", "/certs/priv.key", mux)
	if err != nil {
		log.Fatal("HTTPS Server error: ", err)
	}
}

func redirectToTLS(w http.ResponseWriter, r *http.Request) {
	// Redirect to the same host but with HTTPS
	log.Printf("[Redirect] redirecting to https")
	var redirectCode int
	if r.Method == http.MethodGet {
		redirectCode = http.StatusMovedPermanently
	} else {
		redirectCode = http.StatusPermanentRedirect
	}
	http.Redirect(w, r, fmt.Sprintf("https://%s%s?%s", r.Host, r.URL.Path, r.URL.RawQuery), redirectCode)
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Request] %s %s", r.Method, r.URL.String())
	subdomain := strings.Split(r.Host, ".")[0]
	routeMap, ok := subdomainRouteMap[subdomain]
	if !ok {
		http.Error(w, fmt.Sprintf("no matching route for subdomain %s", subdomain), http.StatusNotFound)
		return
	}
	for _, route := range routeMap {
		if strings.HasPrefix(r.URL.Path, route.Path) {
			realPath := strings.TrimPrefix(r.URL.Path, route.Path)
			origHost := r.Host
			r.URL.Path = realPath
			log.Printf("[Route] %s -> %s%s ", origHost, route.Url.String(), route.Path)
			proxyServer := httputil.NewSingleHostReverseProxy(&route.Url)
			proxyServer.Transport = transport
			proxyServer.ServeHTTP(w, r)
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
		if config.Port == "" {
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
			subdomainRouteMap[alias] = make([]Route, 0)
		}
		url, err := url.Parse(fmt.Sprintf("%s://%s:%s", config.Scheme, config.Host, config.Port))
		if err != nil {
			log.Fatal(err)
		}
		route := Route{Url: *url, Path: config.Path}
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
}
