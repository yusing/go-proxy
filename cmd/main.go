package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal"
	"github.com/yusing/go-proxy/internal/api"
	"github.com/yusing/go-proxy/internal/api/v1/query"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	R "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/server"
	"github.com/yusing/go-proxy/pkg"
)

func main() {
	args := common.GetArgs()

	if args.Command == common.CommandSetup {
		internal.Setup()
		return
	}

	l := logrus.WithField("module", "main")

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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	autocert := cfg.GetAutoCertProvider()
	if autocert != nil {
		if err := autocert.Setup(); err != nil {
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

	// wait for signal
	<-sig

	// grafully shutdown
	logrus.Info("shutting down")
	common.CancelGlobalContext()
	common.GlobalContextWait(time.Second * time.Duration(cfg.Value().TimeoutShutdown))
}

func prepareDirectory(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0o755); err != nil {
			logrus.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}
}

func printJSON(obj any) {
	j, err := E.Check(json.MarshalIndent(obj, "", "  "))
	if err != nil {
		logrus.Fatal(err)
	}
	rawLogger := log.New(os.Stdout, "", 0)
	rawLogger.Printf("%s", j) // raw output for convenience using "jq"
}
