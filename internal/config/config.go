package config

import (
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/autocert"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/route"
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
	logger     = logrus.WithField("module", "config")
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
			logger.Error(err)
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
		logger.Warn(cfgRenameWarn)
		return
	case events.ActionFileDeleted:
		logger.Warn(cfgDeleteWarn)
		return
	}

	if err := Reload(); err != nil {
		logger.Error(err)
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
	b := E.NewBuilder("errors starting providers")
	cfg.providers.RangeAllParallel(func(_ string, p *proxy.Provider) {
		b.Add(p.Start(cfg.task.Subtask(p.String())))
	})

	if b.HasError() {
		logger.Error(b.Build())
	}
}

func (cfg *Config) load() (res E.Error) {
	b := E.NewBuilder("errors loading config")
	defer b.To(&res)

	logger.Debug("loading config")
	defer logger.Debug("loaded config")

	data, err := E.Check(os.ReadFile(common.ConfigPath))
	if err != nil {
		b.Add(E.FailWith("read config", err))
		logrus.Fatal(b.Build())
	}

	if !common.NoSchemaValidation {
		if err = Validate(data); err != nil {
			b.Add(E.FailWith("schema validation", err))
			logrus.Fatal(b.Build())
		}
	}

	model := types.DefaultConfig()
	if err := E.From(yaml.Unmarshal(data, model)); err != nil {
		b.Add(E.FailWith("parse config", err))
		logrus.Fatal(b.Build())
	}

	// errors are non fatal below
	b.Add(cfg.initAutoCert(&model.AutoCert))
	b.Add(cfg.loadProviders(&model.Providers))

	cfg.value = model
	route.SetFindMuxDomains(model.MatchDomains)
	return
}

func (cfg *Config) initAutoCert(autocertCfg *types.AutoCertConfig) (err E.Error) {
	if cfg.autocertProvider != nil {
		return
	}

	logger.Debug("initializing autocert")
	defer logger.Debug("initialized autocert")

	cfg.autocertProvider, err = autocert.NewConfig(autocertCfg).GetProvider()
	if err != nil {
		err = E.FailWith("autocert provider", err)
	}
	return
}

func (cfg *Config) loadProviders(providers *types.ProxyProviders) (outErr E.Error) {
	subtask := cfg.task.Subtask("load providers")
	defer subtask.Finish("done")

	errs := E.NewBuilder("errors loading providers")
	results := E.NewBuilder("loaded providers")
	defer errs.To(&outErr)

	for _, filename := range providers.Files {
		p, err := proxy.NewFileProvider(filename)
		if err != nil {
			errs.Add(err)
			continue
		}
		cfg.providers.Store(p.GetName(), p)
		errs.Add(p.LoadRoutes().Subject(filename))
		results.Addf("%d routes from %s", p.NumRoutes(), p.String())
	}
	for name, dockerHost := range providers.Docker {
		p, err := proxy.NewDockerProvider(name, dockerHost)
		if err != nil {
			errs.Add(err.Subjectf("%s (%s)", name, dockerHost))
			continue
		}
		cfg.providers.Store(p.GetName(), p)
		errs.Add(p.LoadRoutes().Subject(p.GetName()))
		results.Addf("%d routes from %s", p.NumRoutes(), p.String())
	}
	logger.Info(results.Build())
	return
}
