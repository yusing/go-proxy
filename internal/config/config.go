package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yusing/go-proxy/internal/autocert"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/entrypoint"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/notif"
	proxy "github.com/yusing/go-proxy/internal/route/provider"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
	"gopkg.in/yaml.v3"
)

type Config struct {
	value            *types.Config
	providers        F.Map[string, *proxy.Provider]
	autocertProvider *autocert.Provider
	task             task.Task
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

func GetInstance() *Config {
	return instance
}

func newConfig() *Config {
	return &Config{
		value:     types.DefaultConfig(),
		providers: F.NewMapOf[string, *proxy.Provider](),
		task:      task.GlobalTask("config"),
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

func Validate(data []byte) E.Error {
	return U.ValidateYaml(U.GetSchema(common.ConfigSchemaPath), data)
}

func MatchDomains() []string {
	return instance.value.MatchDomains
}

func WatchChanges() {
	task := task.GlobalTask("Config watcher")
	eventQueue := events.NewEventQueue(
		task,
		configEventFlushInterval,
		OnConfigChange,
		func(err E.Error) {
			E.LogError("config reload error", err, &logger)
		},
	)
	eventQueue.Start(cfgWatcher.Events(task.Context()))
}

func OnConfigChange(flushTask task.Task, ev []events.Event) {
	defer flushTask.Finish("config reload complete")

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
		return err
	}

	// cancel all current subtasks -> wait
	// -> replace config -> start new subtasks
	instance.task.Finish("config changed")
	instance.task.Wait()
	*instance = *newCfg
	instance.StartProxyProviders()
	return nil
}

func Value() types.Config {
	return *instance.value
}

func GetAutoCertProvider() *autocert.Provider {
	return instance.autocertProvider
}

func (cfg *Config) Task() task.Task {
	return cfg.task
}

func (cfg *Config) StartProxyProviders() {
	errs := cfg.providers.CollectErrorsParallel(
		func(_ string, p *proxy.Provider) error {
			subtask := cfg.task.Subtask(p.String())
			return p.Start(subtask)
		})

	if err := E.Join(errs...); err != nil {
		E.LogError("route provider errors", err, &logger)
	}
}

func (cfg *Config) load() E.Error {
	const errMsg = "config load error"

	data, err := os.ReadFile(common.ConfigPath)
	if err != nil {
		E.LogFatal(errMsg, err, &logger)
	}

	if !common.NoSchemaValidation {
		if err := Validate(data); err != nil {
			E.LogFatal(errMsg, err, &logger)
		}
	}

	model := types.DefaultConfig()
	if err := E.From(yaml.Unmarshal(data, model)); err != nil {
		E.LogFatal(errMsg, err, &logger)
	}

	// errors are non fatal below
	errs := E.NewBuilder(errMsg)
	errs.Add(cfg.initNotification(model.Providers.Notification))
	errs.Add(cfg.initAutoCert(&model.AutoCert))
	errs.Add(cfg.loadRouteProviders(&model.Providers))

	cfg.value = model
	for i, domain := range model.MatchDomains {
		if !strings.HasPrefix(domain, ".") {
			model.MatchDomains[i] = "." + domain
		}
	}
	entrypoint.SetFindRouteDomains(model.MatchDomains)
	return errs.Error()
}

func (cfg *Config) initNotification(notifCfgMap types.NotificationConfigMap) (err E.Error) {
	if len(notifCfgMap) == 0 {
		return
	}
	errs := E.NewBuilder("notification providers load errors")
	for name, notifCfg := range notifCfgMap {
		_, err := notif.RegisterProvider(cfg.task.Subtask(name), notifCfg)
		errs.Add(err)
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
	subtask := cfg.task.Subtask("load route providers")
	defer subtask.Finish("done")

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
