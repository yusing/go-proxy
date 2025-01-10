package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yusing/go-proxy/internal"
	"github.com/yusing/go-proxy/internal/api"
	"github.com/yusing/go-proxy/internal/api/v1/auth"
	"github.com/yusing/go-proxy/internal/api/v1/query"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/entrypoint"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/metrics"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/net/http/server"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/pkg"
)

func main() {
	args := common.GetArgs()

	switch args.Command {
	case common.CommandSetup:
		internal.Setup()
		return
	case common.CommandReload:
		if err := query.ReloadServer(); err != nil {
			E.LogFatal("server reload error", err)
		}
		logging.Info().Msg("ok")
		return
	case common.CommandListIcons:
		icons, err := internal.ListAvailableIcons()
		if err != nil {
			log.Fatal(err)
		}
		printJSON(icons)
		return
	case common.CommandListRoutes:
		routes, err := query.ListRoutes()
		if err != nil {
			log.Printf("failed to connect to api server: %s", err)
			log.Printf("falling back to config file")
		} else {
			printJSON(routes)
			return
		}
	case common.CommandDebugListMTrace:
		trace, err := query.ListMiddlewareTraces()
		if err != nil {
			log.Fatal(err)
		}
		printJSON(trace)
		return
	}

	if args.Command == common.CommandStart {
		logging.Info().Msgf("GoDoxy version %s", pkg.GetVersion())
		logging.Trace().Msg("trace enabled")
		// logging.AddHook(notif.GetDispatcher())
	} else {
		logging.DiscardLogger()
	}

	if args.Command == common.CommandValidate {
		data, err := os.ReadFile(common.ConfigPath)
		if err == nil {
			err = config.Validate(data)
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

	var cfg *config.Config
	var err E.Error
	if cfg, err = config.Load(); err != nil {
		E.LogWarn("errors in config", err)
	}

	switch args.Command {
	case common.CommandListRoutes:
		cfg.StartProxyProviders()
		printJSON(config.RoutesByAlias())
		return
	case common.CommandListConfigs:
		printJSON(config.Value())
		return
	case common.CommandDebugListEntries:
		printJSON(config.DumpEntries())
		return
	case common.CommandDebugListProviders:
		printJSON(config.DumpProviders())
		return
	}

	if common.APIJWTSecret == nil {
		logging.Warn().Msg("API JWT secret is empty, authentication is disabled")
	}

	cfg.StartProxyProviders()
	config.WatchChanges()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	autocert := config.GetAutoCertProvider()
	if autocert != nil {
		if err := autocert.Setup(); err != nil {
			E.LogFatal("autocert setup error", err)
		}
	} else {
		logging.Info().Msg("autocert not configured")
	}

	server.StartServer(server.Options{
		Name:         "proxy",
		CertProvider: autocert,
		HTTPAddr:     common.ProxyHTTPAddr,
		HTTPSAddr:    common.ProxyHTTPSAddr,
		Handler:      http.HandlerFunc(entrypoint.Handler),
	})

	// Initialize authentication providers
	if err := auth.Initialize(); err != nil {
		logging.Warn().Err(err).Msg("Failed to initialize authentication providers")
	}

	server.StartServer(server.Options{
		Name:         "api",
		CertProvider: autocert,
		HTTPAddr:     common.APIHTTPAddr,
		Handler:      api.NewHandler(),
	})

	if common.PrometheusEnabled {
		server.StartServer(server.Options{
			Name:         "metrics",
			CertProvider: autocert,
			HTTPAddr:     common.MetricsHTTPAddr,
			Handler:      metrics.NewHandler(),
		})
	}

	// wait for signal
	<-sig

	// gracefully shutdown
	logging.Info().Msg("shutting down")
	_ = task.GracefulShutdown(time.Second * time.Duration(config.Value().TimeoutShutdown))
}

func prepareDirectory(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0o755); err != nil {
			logging.Fatal().Msgf("failed to create directory %s: %v", dir, err)
		}
	}
}

func printJSON(obj any) {
	j, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		logging.Fatal().Err(err).Send()
	}
	rawLogger := log.New(os.Stdout, "", 0)
	rawLogger.Print(string(j)) // raw output for convenience using "jq"
}
