package main

import (
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func main() {
	var err error

	// flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		DisableColors: false,
		FullTimestamp: true,
	})
	InitFSWatcher()
	InitDockerWatcher()

	cfg := NewConfig()
	cfg.MustLoad()
	cfg.StartProviders()
	cfg.WatchChanges()

	var certAvailable = utils.fileOK(certPath) && utils.fileOK(keyPath)

	go func() {
		log.Info("starting http server on port 80")
		if certAvailable && redirectHTTP {
			err = http.ListenAndServe(":80", http.HandlerFunc(redirectToTLS))
		} else {
			err = http.ListenAndServe(":80", http.HandlerFunc(httpProxyHandler))
		}
		if err != nil {
			log.Fatal("HTTP server error: ", err)
		}
	}()
	go func() {
		log.Infof("starting http panel on port 8080")
		err = http.ListenAndServe(":8080", http.HandlerFunc(panelHandler))
		if err != nil {
			log.Warning("HTTP panel error: ", err)
		}
	}()

	if certAvailable {
		go func() {
			log.Info("starting https server on port 443")
			err = http.ListenAndServeTLS(":443", certPath, keyPath, http.HandlerFunc(httpProxyHandler))
			if err != nil {
				log.Fatal("https server error: ", err)
			}
		}()
		go func() {
			log.Info("starting https panel on port 8443")
			err := http.ListenAndServeTLS(":8443", certPath, keyPath, http.HandlerFunc(panelHandler))
			if err != nil {
				log.Warning("http panel error: ", err)
			}
		}()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	<-sig
	cfg.StopWatching()
	cfg.StopProviders()
	close(fsWatcherStop)
	close(dockerWatcherStop)
}
