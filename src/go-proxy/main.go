package main

import (
	"flag"
	"net/http"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

func main() {
	var err error
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		glog.Fatal(err)
	}

	buildRoutes()
	glog.Infof("[Build] built %v reverse proxies", CountRoutes())
	BeginListenStreams()

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
				glog.Infof("[Event] %s %s caused rebuild", msg.Action, msg.Actor.Attributes["name"])
				EndListenStreams()
				buildRoutes()
				glog.Infof("[Build] rebuilt %v reverse proxies", CountRoutes())
				BeginListenStreams()
			case err := <-errChan:
				glog.Infof("[Event] %s", err)
				msgChan, errChan = dockerClient.Events(context.Background(), types.EventsOptions{Filters: filter})
			}
		}
	}()

	go func() {
		for range time.Tick(100 * time.Millisecond) {
			glog.Flush()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpProxyHandler)

	go func() {
		glog.Infoln("Starting HTTP server on port 80")
		err := http.ListenAndServe(":80", http.HandlerFunc(redirectToTLS))
		if err != nil {
			glog.Fatal("HTTP server error", err)
		}
	}()
	go func() {
		glog.Infoln("Starting HTTPS panel on port 8443")
		err := http.ListenAndServeTLS(":8443", "/certs/cert.crt", "/certs/priv.key", http.HandlerFunc(panelHandler))
		if err != nil {
			glog.Fatal("HTTP server error", err)
		}
	}()
	glog.Infoln("Starting HTTPS server on port 443")
	err = http.ListenAndServeTLS(":443", "/certs/cert.crt", "/certs/priv.key", mux)
	if err != nil {
		glog.Fatal("HTTPS Server error: ", err)
	}
}
