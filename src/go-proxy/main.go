package go_proxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"

	mapset "github.com/deckarep/golang-set/v2"
)

var panelRoute = mapset.NewSet(Route{Url: &url.URL{Scheme: "http", Host: "localhost:81", Path: "/"}, Path: "/"})

// TODO: default + per proxy
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
	go func() {
		log.Println("Starting HTTP panel on port 81")
		err := http.ListenAndServe(":81", http.HandlerFunc(panelHandler))
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
	for route := range routeMap.Iter() {
		if strings.HasPrefix(r.URL.Path, route.Path) {
			realPath := strings.TrimPrefix(r.URL.Path, route.Path)
			origHost := r.Host
			r.URL.Path = realPath
			log.Printf("[Route] %s -> %s%s ", origHost, route.Url.String(), route.Path)
			proxyServer := httputil.NewSingleHostReverseProxy(route.Url)
			proxyServer.Transport = transport
			proxyServer.ServeHTTP(w, r)
			return
		}
	}
	http.Error(w, fmt.Sprintf("no matching route for path %s for subdomain %s", r.URL.Path, subdomain), http.StatusNotFound)
}
