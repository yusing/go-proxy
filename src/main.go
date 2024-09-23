package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/api"
	apiUtils "github.com/yusing/go-proxy/api/v1/utils"
	"github.com/yusing/go-proxy/common"
	"github.com/yusing/go-proxy/config"
	"github.com/yusing/go-proxy/docker"
	"github.com/yusing/go-proxy/docker/idlewatcher"
	E "github.com/yusing/go-proxy/error"
	R "github.com/yusing/go-proxy/route"
	"github.com/yusing/go-proxy/server"
	F "github.com/yusing/go-proxy/utils/functional"
)

func main() {
	args := common.GetArgs()

	if args.Command == common.CommandSetup {
		Setup()
		return
	}

	l := logrus.WithField("module", "main")
	onShutdown := F.NewSlice[func()]()

	if common.IsDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if args.Command != common.CommandStart {
		logrus.SetOutput(io.Discard)
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableSorting:         true,
			DisableLevelTruncation: true,
			FullTimestamp:          true,
			ForceColors:            true,
			TimestampFormat:        "01-02 15:04:05",
		})
	}

	if args.Command == common.CommandReload {
		if err := apiUtils.ReloadServer(); err.HasError() {
			log.Fatal(err)
		}
		log.Print("ok")
		return
	}

	// exit if only validate config
	if args.Command == common.CommandValidate {
		data, err := os.ReadFile(common.ConfigPath)
		if err == nil {
			err = config.Validate(data).Error()
		}
		if err != nil {
			log.Fatal("config error: ", err)
		}
		log.Print("config OK")
		return
	}

	cfg, err := config.Load()
	if err.IsFatal() {
		log.Fatal(err)
	}

	switch args.Command {
	case common.CommandListConfigs:
		printJSON(cfg.Value())
		return
	case common.CommandListRoutes:
		printJSON(cfg.RoutesByAlias())
		return
	case common.CommandDebugListEntries:
		printJSON(cfg.DumpEntries())
		return
	case common.CommandDebugListProviders:
		printJSON(cfg.DumpProviders())
		return
	}

	cfg.StartProxyProviders()

	if err.HasError() {
		l.Warn(err)
	}

	cfg.WatchChanges()

	onShutdown.Add(docker.CloseAllClients)
	onShutdown.Add(cfg.Dispose)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	autocert := cfg.GetAutoCertProvider()

	if autocert != nil {
		ctx, cancel := context.WithCancel(context.Background())
		if err = autocert.Setup(ctx); err != nil && err.IsWarning() {
			cancel()
			l.Warn(err)
		} else if err.IsFatal() {
			l.Fatal(err)
		} else {
			onShutdown.Add(cancel)
		}
	} else {
		l.Info("autocert not configured")
	}

	proxyServer := server.InitProxyServer(server.Options{
		Name:            "proxy",
		CertProvider:    autocert,
		HTTPAddr:        common.ProxyHTTPAddr,
		HTTPSAddr:       common.ProxyHTTPSAddr,
		Handler:         http.HandlerFunc(R.ProxyHandler),
		RedirectToHTTPS: cfg.Value().RedirectToHTTPS,
	})
	apiServer := server.InitAPIServer(server.Options{
		Name:            "api",
		CertProvider:    autocert,
		HTTPAddr:        common.APIHTTPAddr,
		Handler:         api.NewHandler(cfg),
		RedirectToHTTPS: cfg.Value().RedirectToHTTPS,
	})

	proxyServer.Start()
	apiServer.Start()
	onShutdown.Add(proxyServer.Stop)
	onShutdown.Add(apiServer.Stop)

	go idlewatcher.Start()
	onShutdown.Add(idlewatcher.Stop)

	// wait for signal
	<-sig

	// grafully shutdown
	logrus.Info("shutting down")
	done := make(chan struct{}, 1)

	var wg sync.WaitGroup
	wg.Add(onShutdown.Size())
	onShutdown.ForEach(func(f func()) {
		go func() {
			l.Debugf("waiting for %s to complete...", funcName(f))
			f()
			l.Debugf("%s done", funcName(f))
			wg.Done()
		}()
	})
	go func() {
		wg.Wait()
		close(done)
	}()

	timeout := time.After(time.Duration(cfg.Value().TimeoutShutdown) * time.Second)
	select {
	case <-done:
		logrus.Info("shutdown complete")
	case <-timeout:
		logrus.Info("timeout waiting for shutdown")
		onShutdown.ForEach(func(f func()) {
			l.Warnf("%s() is still running", funcName(f))
		})
	}
}

func funcName(f func()) string {
	parts := strings.Split(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), "/go-proxy/")
	return parts[len(parts)-1]
}

func printJSON(obj any) {
	j, err := E.Check(json.Marshal(obj))
	if err.HasError() {
		logrus.Fatal(err)
	}
	rawLogger := log.New(os.Stdout, "", 0)
	rawLogger.Printf("%s", j) // raw output for convenience using "jq"
}
