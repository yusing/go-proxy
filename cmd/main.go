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
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal"
	"github.com/yusing/go-proxy/internal/api"
	"github.com/yusing/go-proxy/internal/api/v1/query"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/docker"
	"github.com/yusing/go-proxy/internal/docker/idlewatcher"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	R "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/server"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/pkg"
)

func main() {
	args := common.GetArgs()

	if args.Command == common.CommandSetup {
		internal.Setup()
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
			DisableSorting:  true,
			FullTimestamp:   true,
			ForceColors:     true,
			TimestampFormat: "01-02 15:04:05",
		})
		logrus.Infof("go-proxy version %s", pkg.GetVersion())
	}

	if args.Command == common.CommandReload {
		if err := query.ReloadServer(); err != nil {
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

	for _, dir := range common.RequiredDirectories {
		prepareDirectory(dir)
	}

	middleware.LoadComposeFiles()

	if err := config.Load(); err != nil {
		logrus.Warn(err)
	}
	cfg := config.GetInstance()

	switch args.Command {
	case common.CommandListConfigs:
		printJSON(cfg.Value())
		return
	case common.CommandListRoutes:
		routes, err := query.ListRoutes()
		if err != nil {
			log.Printf("failed to connect to api server: %s", err)
			log.Printf("falling back to config file")
			printJSON(cfg.RoutesByAlias())
		} else {
			printJSON(routes)
		}
		return
	case common.CommandListIcons:
		icons, err := internal.ListAvailableIcons()
		if err != nil {
			log.Fatal(err)
		}
		printJSON(icons)
		return
	case common.CommandDebugListEntries:
		printJSON(cfg.DumpEntries())
		return
	case common.CommandDebugListProviders:
		printJSON(cfg.DumpProviders())
		return
	case common.CommandDebugListMTrace:
		trace, err := query.ListMiddlewareTraces()
		if err != nil {
			log.Fatal(err)
		}
		printJSON(trace)
	}

	cfg.StartProxyProviders()
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
		onShutdown.Add(cancel)
		if err := autocert.Setup(ctx); err != nil {
			l.Fatal(err)
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
	currentIdx := 0

	go func() {
		onShutdown.ForEach(func(f func()) {
			l.Debugf("waiting for %s to complete...", funcName(f))
			f()
			currentIdx++
			l.Debugf("%s done", funcName(f))
		})
		close(done)
	}()

	timeout := time.After(time.Duration(cfg.Value().TimeoutShutdown) * time.Second)
	select {
	case <-done:
		logrus.Info("shutdown complete")
	case <-timeout:
		logrus.Info("timeout waiting for shutdown")
		for i := currentIdx; i < onShutdown.Size(); i++ {
			l.Warnf("%s() is still running", funcName(onShutdown.Get(i)))
		}
	}
}

func prepareDirectory(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			logrus.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}
}

func funcName(f func()) string {
	parts := strings.Split(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), "/go-proxy/")
	return parts[len(parts)-1]
}

func printJSON(obj any) {
	j, err := E.Check(json.MarshalIndent(obj, "", "  "))
	if err != nil {
		logrus.Fatal(err)
	}
	rawLogger := log.New(os.Stdout, "", 0)
	rawLogger.Printf("%s", j) // raw output for convenience using "jq"
}
