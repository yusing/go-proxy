package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yusing/go-proxy/internal/api"
	"github.com/yusing/go-proxy/internal/api/v1/auth"
	"github.com/yusing/go-proxy/internal/autocert"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/entrypoint"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/metrics"
	"github.com/yusing/go-proxy/internal/net/http/server"
	"github.com/yusing/go-proxy/internal/notif"
	proxy "github.com/yusing/go-proxy/internal/route/provider"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type Config struct {
	value            *types.Config
	providers        F.Map[string, *proxy.Provider]
	autocertProvider *autocert.Provider
	entrypoint       *entrypoint.Entrypoint

	task *task.Task
}

var (
	instance   *Config
	cfgWatcher watcher.Watcher
	logger     = logging.With().Str("module", "config").Logger()
	reloadMu   sync.Mutex
)

const configEventFlushInterval = 500 * time.Millisecond

const (
	cfgRenameWarn = `Config file renamed, not reloading.
Make sure you rename it back before next time you start.`
	cfgDeleteWarn = `Config file deleted, not reloading.
You may run "ls-config" to show or dump the current config.`
)

var Validate = types.Validate

func GetInstance() *Config {
	return instance
}

func newConfig() *Config {
	return &Config{
		value:      types.DefaultConfig(),
		providers:  F.NewMapOf[string, *proxy.Provider](),
		entrypoint: entrypoint.NewEntrypoint(),
		task:       task.RootTask("config", false),
	}
}

func Load() (*Config, E.Error) {
	if instance != nil {
		return instance, nil
	}
	instance = newConfig()
	cfgWatcher = watcher.NewConfigFileWatcher(common.ConfigFileName)
	return instance, instance.load()
}

func MatchDomains() []string {
	return instance.value.MatchDomains
}

func WatchChanges() {
	t := task.RootTask("config_watcher", true)
	eventQueue := events.NewEventQueue(
		t,
		configEventFlushInterval,
		OnConfigChange,
		func(err E.Error) {
			E.LogError("config reload error", err, &logger)
		},
	)
	eventQueue.Start(cfgWatcher.Events(t.Context()))
}

func OnConfigChange(ev []events.Event) {
	// no matter how many events during the interval
	// just reload once and check the last event
	switch ev[len(ev)-1].Action {
	case events.ActionFileRenamed:
		logger.Warn().Msg(cfgRenameWarn)
		return
	case events.ActionFileDeleted:
		logger.Warn().Msg(cfgDeleteWarn)
		return
	}

	if err := Reload(); err != nil {
		logger.Warn().Msg("using last config")
		// recovered in event queue
		panic(err)
	}
}

func Reload() E.Error {
	// avoid race between config change and API reload request
	reloadMu.Lock()
	defer reloadMu.Unlock()

	newCfg := newConfig()
	err := newCfg.load()
	if err != nil {
		newCfg.task.Finish(err)
		return err
	}

	// cancel all current subtasks -> wait
	// -> replace config -> start new subtasks
	instance.task.Finish("config changed")
	instance = newCfg
	instance.Start()
	return nil
}

func (cfg *Config) Value() *types.Config {
	return instance.value
}

func (cfg *Config) Reload() E.Error {
	return Reload()
}

func (cfg *Config) AutoCertProvider() *autocert.Provider {
	return instance.autocertProvider
}

func (cfg *Config) Task() *task.Task {
	return cfg.task
}

func (cfg *Config) Start() {
	cfg.StartAutoCert()
	cfg.StartProxyProviders()
	cfg.StartServers()
}

func (cfg *Config) StartAutoCert() {
	autocert := cfg.autocertProvider
	if autocert == nil {
		logging.Info().Msg("autocert not configured")
		return
	}

	if err := autocert.Setup(); err != nil {
		E.LogFatal("autocert setup error", err)
	} else {
		autocert.ScheduleRenewal(cfg.task)
	}
}

func (cfg *Config) StartProxyProviders() {
	errs := cfg.providers.CollectErrorsParallel(
		func(_ string, p *proxy.Provider) error {
			return p.Start(cfg.task)
		})

	if err := E.Join(errs...); err != nil {
		E.LogError("route provider errors", err, &logger)
	}
}

func (cfg *Config) StartServers() {
	server.StartServer(cfg.task, server.Options{
		Name:         "proxy",
		CertProvider: cfg.AutoCertProvider(),
		HTTPAddr:     common.ProxyHTTPAddr,
		HTTPSAddr:    common.ProxyHTTPSAddr,
		Handler:      cfg.entrypoint,
	})
	server.StartServer(cfg.task, server.Options{
		Name:         "api",
		CertProvider: cfg.AutoCertProvider(),
		HTTPAddr:     common.APIHTTPAddr,
		Handler:      api.NewHandler(cfg),
	})
	if common.PrometheusEnabled {
		server.StartServer(cfg.task, server.Options{
			Name:         "metrics",
			CertProvider: cfg.AutoCertProvider(),
			HTTPAddr:     common.MetricsHTTPAddr,
			Handler:      metrics.NewHandler(),
		})
	}
}

func (cfg *Config) load() E.Error {
	const errMsg = "config load error"

	data, err := os.ReadFile(common.ConfigPath)
	if err != nil {
		E.LogFatal(errMsg, err, &logger)
	}

	model := types.DefaultConfig()
	if err := utils.DeserializeYAML(data, model); err != nil {
		E.LogFatal(errMsg, err, &logger)
	}

	// errors are non fatal below
	errs := E.NewBuilder(errMsg)
	errs.Add(cfg.entrypoint.SetMiddlewares(model.Entrypoint.Middlewares))
	errs.Add(cfg.entrypoint.SetAccessLogger(cfg.task, model.Entrypoint.AccessLog))
	errs.Add(cfg.initNotification(model.Providers.Notification))
	errs.Add(cfg.initAutoCert(model.AutoCert))
	errs.Add(cfg.loadRouteProviders(&model.Providers))

	cfg.value = model
	for i, domain := range model.MatchDomains {
		if !strings.HasPrefix(domain, ".") {
			model.MatchDomains[i] = "." + domain
		}
	}
	cfg.entrypoint.SetFindRouteDomains(model.MatchDomains)

	return errs.Error()
}

func (cfg *Config) initNotification(notifCfg []types.NotificationConfig) (err E.Error) {
	if len(notifCfg) == 0 {
		return
	}
	dispatcher := notif.StartNotifDispatcher(cfg.task)
	errs := E.NewBuilder("notification providers load errors")
	for i, notifier := range notifCfg {
		_, err := dispatcher.RegisterProvider(notifier)
		if err == nil {
			continue
		}
		errs.Add(err.Subjectf("[%d]", i))
	}
	return errs.Error()
}

func (cfg *Config) initAutoCert(autocertCfg *types.AutoCertConfig) (err E.Error) {
	if cfg.autocertProvider != nil {
		return
	}

	cfg.autocertProvider, err = autocert.NewConfig(autocertCfg).GetProvider()
	return
}

func (cfg *Config) loadRouteProviders(providers *types.Providers) E.Error {
	errs := E.NewBuilder("route provider errors")
	results := E.NewBuilder("loaded route providers")

	lenLongestName := 0
	for _, filename := range providers.Files {
		p, err := proxy.NewFileProvider(filename)
		if err != nil {
			errs.Add(E.PrependSubject(filename, err))
			continue
		}
		cfg.providers.Store(p.GetName(), p)
		if len(p.GetName()) > lenLongestName {
			lenLongestName = len(p.GetName())
		}
	}
	for name, dockerHost := range providers.Docker {
		p, err := proxy.NewDockerProvider(name, dockerHost)
		if err != nil {
			errs.Add(E.PrependSubject(name, err))
			continue
		}
		cfg.providers.Store(p.GetName(), p)
		if len(p.GetName()) > lenLongestName {
			lenLongestName = len(p.GetName())
		}
	}
	cfg.providers.RangeAllParallel(func(_ string, p *proxy.Provider) {
		if err := p.LoadRoutes(); err != nil {
			errs.Add(err.Subject(p.String()))
		}
		results.Addf("%-"+strconv.Itoa(lenLongestName)+"s %d routes", p.GetName(), p.NumRoutes())
	})
	logger.Info().Msg(results.String())
	return errs.Error()
}
