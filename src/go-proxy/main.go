package main

import (
	"log"
	"net/http"
	"runtime"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func main() {
	var err error
	runtime.GOMAXPROCS(runtime.NumCPU())

	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	buildRoutes()
	log.Printf("[Build] built %v reverse proxies", countRoutes())
	beginListenStreams()

	go func() {
		filter := filters.NewArgs(
			filters.Arg("type", "container"),
			filters.Arg("event", "start"),
			filters.Arg("event", "die"), // stop seems like triggering die
			// filters.Arg("event", "stop"),
		)
		msgChan, errChan := dockerClient.Events(context.Background(), types.EventsOptions{Filters: filter})

		for {
			select {
			case msg := <-msgChan:
				// TODO: handle actor only
				log.Printf("[Event] %s %s caused rebuild", msg.Action, msg.Actor.Attributes["name"])
				endListenStreams()
				buildRoutes()
				log.Printf("[Build] rebuilt %v reverse proxies", countRoutes())
				beginListenStreams()
			case err := <-errChan:
				log.Printf("[Event] %s", err)
				msgChan, errChan = dockerClient.Events(context.Background(), types.EventsOptions{Filters: filter})
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpProxyHandler)

	go func() {
		log.Println("Starting HTTP server on port 80")
		err := http.ListenAndServe(":80", http.HandlerFunc(redirectToTLS))
		if err != nil {
			log.Fatal("HTTP server error", err)
		}
	}()
	go func() {
		log.Println("Starting HTTPS panel on port 8443")
		err := http.ListenAndServeTLS(":8443", "/certs/cert.crt", "/certs/priv.key", http.HandlerFunc(panelHandler))
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
