package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/sirupsen/logrus"
)

var cfg Config

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var verifyOnly bool 
	flag.BoolVar(&verifyOnly, "verify", false, "verify config without starting server")
	flag.Parse()

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		DisableColors: false,
		FullTimestamp: true,
		TimestampFormat: "01-02 15:04:05",
	})

	cfg = NewConfig(configPath)
	cfg.MustLoad()

	if verifyOnly {
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
		RedirectToHTTPS: redirectToHTTPS,
	})
	panelServer = NewServer(ServerOptions{
		Name:            "panel",
		CertProvider:    autoCertProvider,
		HTTPAddr:        ":8080",
		HTTPSAddr:       ":8443",
		Handler:         panelHandler,
		RedirectToHTTPS: redirectToHTTPS,
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
	// cfg.StopWatching()
	StopFSWatcher()
	StopDockerWatcher()
	cfg.StopProviders()
	panelServer.Stop()
	proxyServer.Stop()
}
