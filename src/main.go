package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/api"
	apiUtils "github.com/yusing/go-proxy/api/v1/utils"
	"github.com/yusing/go-proxy/common"
	"github.com/yusing/go-proxy/config"
	"github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	R "github.com/yusing/go-proxy/route"
	"github.com/yusing/go-proxy/server"
	F "github.com/yusing/go-proxy/utils/functional"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	args := common.GetArgs()
	l := logrus.WithField("?", "init")

	if common.IsDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if common.IsRunningAsService {
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableColors:    true,
			DisableTimestamp: true,
			DisableSorting:   true,
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableSorting:  true,
			FullTimestamp:   true,
			TimestampFormat: "01-02 15:04:05",
		})
	}

	if args.Command == common.CommandReload {
		if err := apiUtils.ReloadServer(); err.IsNotNil() {
			l.Fatal(err)
		}
		return
	}

	onShutdown := F.NewSlice[func()]()

	// exit if only validate config
	if args.Command == common.CommandValidate {
		var err E.NestedError
		data, err := E.Check(os.ReadFile(common.ConfigPath))
		if err.IsNotNil() {
			l.WithError(err).Fatalf("config error")
		}
		if err = config.Validate(data); err.IsNotNil() {
			l.WithError(err).Fatalf("config error")
		}
		l.Printf("config OK")
		return
	}

	cfg, err := config.New()
	if err.IsNotNil() {
		l.Fatalf("config error: %s", err)
	}

	onShutdown.Add(func() {
		docker.CloseAllClients()
		cfg.Dispose()
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	autocert := cfg.GetAutoCertProvider()

	if autocert != nil {
		err = autocert.LoadCert()

		if err.IsNotNil() {
			l.Error(err)
			l.Info("Now attempting to obtain a new certificate...")
			if err = autocert.ObtainCert(); err.IsNotNil() {
				ctx, certRenewalCancel := context.WithCancel(context.Background())
				go autocert.ScheduleRenewal(ctx)
				onShutdown.Add(certRenewalCancel)
			} else {
				l.Warn(err)
			}
		} else {
			for name, expiry := range autocert.GetExpiries() {
				l.Infof("certificate %q: expire on %s", name, expiry)
			}
		}
	}

	proxyServer := server.InitProxyServer(server.Options{
		Name:            "proxy",
		CertProvider:    autocert,
		HTTPPort:        common.ProxyHTTPPort,
		HTTPSPort:       common.ProxyHTTPSPort,
		Handler:         http.HandlerFunc(R.ProxyHandler),
		RedirectToHTTPS: cfg.Value().RedirectToHTTPS,
	})
	apiServer := server.InitAPIServer(server.Options{
		Name:            "api",
		CertProvider:    autocert,
		HTTPPort:        common.APIHTTPPort,
		Handler:         api.NewHandler(cfg),
		RedirectToHTTPS: cfg.Value().RedirectToHTTPS,
	})

	proxyServer.Start()
	apiServer.Start()
	onShutdown.Add(proxyServer.Stop)
	onShutdown.Add(apiServer.Stop)

	// wait for signal
	<-sig

	// grafully shutdown
	logrus.Info("shutting down")
	done := make(chan struct{}, 1)

	var wg sync.WaitGroup
	wg.Add(onShutdown.Size())
	onShutdown.ForEach(func(f func()) {
		go func() {
			f()
			wg.Done()
		}()
	})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logrus.Info("shutdown complete")
	case <-time.After(time.Duration(cfg.Value().TimeoutShutdown) * time.Second):
		logrus.Info("timeout waiting for shutdown")
	}
}
