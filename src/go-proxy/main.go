package main

import (
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

var cfg Config

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	args := getArgs()

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		DisableColors:   false,
		FullTimestamp:   true,
		TimestampFormat: "01-02 15:04:05",
	})

	if args.Command == CommandReload {
		err := utils.reloadServer()
		if err != nil {
			logrus.Fatal(err)
		}
		return
	}

	cfg = NewConfig(configPath)
	cfg.MustLoad()

	logrus.Info(cfg.Value())

	if args.Command == CommandValidate {
		logrus.Printf("config OK")
		return
	}

	autoCertProvider, err := cfg.GetAutoCertProvider()

	if err != nil {
		aclog.Warn(err)
		autoCertProvider = nil // TODO: remove, it is expected to be nil if error is not nil, but it is not for now
	}

	var proxyServer *Server
	var panelServer *Server

	if autoCertProvider != nil {
		ok := autoCertProvider.LoadCert()
		if !ok {
			if ne := autoCertProvider.ObtainCert(); ne != nil {
				aclog.Fatal(ne)
			}
		}
		for name, expiry := range autoCertProvider.GetExpiries() {
			aclog.Infof("certificate %q: expire on %v", name, expiry)
		}
		go autoCertProvider.ScheduleRenewal()
	}
	proxyServer = NewServer(ServerOptions{
		Name:            "proxy",
		CertProvider:    autoCertProvider,
		HTTPAddr:        ":80",
		HTTPSAddr:       ":443",
		Handler:         http.HandlerFunc(proxyHandler),
		RedirectToHTTPS: cfg.Value().RedirectToHTTPS,
	})
	panelServer = NewServer(ServerOptions{
		Name:            "panel",
		CertProvider:    autoCertProvider,
		HTTPAddr:        ":8080",
		HTTPSAddr:       ":8443",
		Handler:         panelHandler,
		RedirectToHTTPS: cfg.Value().RedirectToHTTPS,
	})

	proxyServer.Start()
	panelServer.Start()

	InitFSWatcher()

	cfg.StartProviders()
	cfg.WatchChanges()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	<-sig
	logrus.Info("shutting down")
	done := make(chan struct{}, 1)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		StopFSWatcher()
		StopDockerWatcher()
		cfg.StopProviders()
		wg.Done()
	}()
	go func() {
		panelServer.Stop()
		proxyServer.Stop()
		wg.Done()
	}()
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logrus.Info("shutdown complete")
	case <-time.After(cfg.Value().TimeoutShutdown * time.Second):
		logrus.Info("timeout waiting for shutdown")
	}
}
