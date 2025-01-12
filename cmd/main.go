package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yusing/go-proxy/internal"
	"github.com/yusing/go-proxy/internal/api/v1/auth"
	"github.com/yusing/go-proxy/internal/api/v1/query"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/route/routes"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/pkg"
)

var rawLogger = log.New(os.Stdout, "", 0)

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
		rawLogger.Println("ok")
		return
	case common.CommandListIcons:
		icons, err := internal.ListAvailableIcons()
		if err != nil {
			rawLogger.Fatal(err)
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
		printJSON(routes.RoutesByAlias())
		return
	case common.CommandListConfigs:
		printJSON(cfg.Value())
		return
	case common.CommandDebugListEntries:
		printJSON(cfg.DumpEntries())
		return
	case common.CommandDebugListProviders:
		printJSON(cfg.DumpProviders())
		return
	}

	cfg.Start()
	config.WatchChanges()

	if !auth.IsEnabled() {
		logging.Warn().Msg("authentication is disabled, please set API_JWT_SECRET or OIDC_* to enable authentication")
	} else {
		// Initialize authentication providers
		if err := auth.Initialize(); err != nil {
			logging.Fatal().Err(err).Msg("Failed to initialize authentication providers")
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)
	signal.Notify(sig, syscall.SIGHUP)

	// wait for signal
	<-sig

	// gracefully shutdown
	logging.Info().Msg("shutting down")
	_ = task.GracefulShutdown(time.Second * time.Duration(cfg.Value().TimeoutShutdown))
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
	rawLogger.Print(string(j)) // raw output for convenience using "jq"
}
