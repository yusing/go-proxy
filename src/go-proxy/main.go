package main

import (
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	// flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		DisableColors: false,
		FullTimestamp: true,
	})

	cfg := NewConfig()
	cfg.MustLoad()

	autoCertProvider, err := cfg.GetAutoCertProvider()

	if err != nil {
		aclog.Warn(err)
		autoCertProvider = nil
	}

	var httpProxyHandler http.Handler
	var httpPanelHandler http.Handler

	var proxyServer *Server
	var panelServer *Server

	if redirectHTTP {
		httpProxyHandler = http.HandlerFunc(redirectToTLSHandler)
		httpPanelHandler = http.HandlerFunc(redirectToTLSHandler)
	} else {
		httpProxyHandler = http.HandlerFunc(proxyHandler)
		httpPanelHandler = http.HandlerFunc(panelHandler)
	}

	if autoCertProvider != nil {
		ok := autoCertProvider.LoadCert()
		if !ok {
			err := autoCertProvider.ObtainCert()
			if err != nil {
				aclog.Fatal("error obtaining certificate ", err)
			}
		}
		aclog.Infof("certificate will be expired at %v and get renewed", autoCertProvider.GetExpiry())
	}
	proxyServer = NewServer(
		"proxy",
		autoCertProvider,
		":80",
		httpProxyHandler,
		":443",
		http.HandlerFunc(proxyHandler),
	)
	panelServer = NewServer(
		"panel",
		autoCertProvider,
		":8080",
		httpPanelHandler,
		":8443",
		http.HandlerFunc(panelHandler),
	)

	proxyServer.Start()
	panelServer.Start()

	InitFSWatcher()
	InitDockerWatcher()

	cfg.StartProviders()
	cfg.WatchChanges()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	<-sig
	cfg.StopWatching()
	StopFSWatcher()
	StopDockerWatcher()
	cfg.StopProviders()
	panelServer.Stop()
	proxyServer.Stop()
}
