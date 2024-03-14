package main

import (
	"flag"
	"net/http"
	"runtime"
	"time"

	"github.com/golang/glog"
)

func main() {
	var err error

	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	go func() {
		for range time.Tick(100 * time.Millisecond) {
			glog.Flush()
		}
	}()

	if config, err = ReadConfig(); err != nil {
		glog.Fatal("unable to read config: ", err)
	}

	StartAllRoutes()
	go ListenConfigChanges()

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpProxyHandler)

	var certAvailable = utils.fileOK(certPath) && utils.fileOK(keyPath)

	go func() {
		glog.Infoln("starting http server on port 80")
		if certAvailable {
			err = http.ListenAndServe(":80", http.HandlerFunc(redirectToTLS))
		} else {
			err = http.ListenAndServe(":80", mux)
		}
		if err != nil {
			glog.Fatal("HTTP server error", err)
		}
	}()
	go func() {
		glog.Infoln("starting http panel on port 8080")
		err := http.ListenAndServe(":8080", http.HandlerFunc(panelHandler))
		if err != nil {
			glog.Fatal("HTTP server error", err)
		}
	}()

	if certAvailable {
		go func() {
			glog.Infoln("starting https panel on port 8443")
			err := http.ListenAndServeTLS(":8443", certPath, keyPath, http.HandlerFunc(panelHandler))
			if err != nil {
				glog.Fatal("http server error", err)
			}
		}()
		go func() {
			glog.Infoln("starting https server on port 443")
			err = http.ListenAndServeTLS(":443", certPath, keyPath, mux)
			if err != nil {
				glog.Fatal("https server error: ", err)
			}
		}()
	}

	<-make(chan struct{})
}
