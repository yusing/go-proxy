package config

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/autocert"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"

	PR "github.com/yusing/go-proxy/internal/proxy/provider"
	R "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/types"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	W "github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
	"gopkg.in/yaml.v3"
)

type Config struct {
	value            *types.Config
	proxyProviders   F.Map[string, *PR.Provider]
	autocertProvider *autocert.Provider

	l logrus.FieldLogger

	watcher       W.Watcher
	watcherCtx    context.Context
	watcherCancel context.CancelFunc
	reloadReq     chan struct{}
}

var instance *Config

func GetInstance() *Config {
	return instance
}

func Load() E.NestedError {
	if instance != nil {
		return nil
	}
	instance = &Config{
		value:          types.DefaultConfig(),
		proxyProviders: F.NewMapOf[string, *PR.Provider](),
		l:              logrus.WithField("module", "config"),
		watcher:        W.NewConfigFileWatcher(common.ConfigFileName),
		reloadReq:      make(chan struct{}, 1),
	}
	return instance.load()
}

func Validate(data []byte) E.NestedError {
	return U.ValidateYaml(U.GetSchema(common.ConfigSchemaPath), data)
}

func MatchDomains() []string {
	if instance == nil {
		logrus.Panic("config has not been loaded, please check if there is any errors")
	}
	return instance.value.MatchDomains
}

func (cfg *Config) Value() types.Config {
	if cfg == nil {
		logrus.Panic("config has not been loaded, please check if there is any errors")
	}
	return *cfg.value
}

func (cfg *Config) GetAutoCertProvider() *autocert.Provider {
	if instance == nil {
		logrus.Panic("config has not been loaded, please check if there is any errors")
	}
	return cfg.autocertProvider
}

func (cfg *Config) Dispose() {
	if cfg.watcherCancel != nil {
		cfg.watcherCancel()
		cfg.l.Debug("stopped watcher")
	}
	cfg.stopProviders()
}

func (cfg *Config) Reload() (err E.NestedError) {
	cfg.stopProviders()
	err = cfg.load()
	cfg.StartProxyProviders()
	return
}

func (cfg *Config) StartProxyProviders() {
	cfg.controlProviders("start", (*PR.Provider).StartAllRoutes)
}

func (cfg *Config) WatchChanges() {
	cfg.watcherCtx, cfg.watcherCancel = context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-cfg.watcherCtx.Done():
				return
			case <-cfg.reloadReq:
				if err := cfg.Reload(); err.HasError() {
					cfg.l.Error(err)
				}
			}
		}
	}()
	go func() {
		eventCh, errCh := cfg.watcher.Events(cfg.watcherCtx)
		for {
			select {
			case <-cfg.watcherCtx.Done():
				return
			case event := <-eventCh:
				if event.Action == events.ActionFileDeleted || event.Action == events.ActionFileRenamed {
					cfg.l.Error("config file deleted or renamed, ignoring...")
					continue
				} else {
					cfg.reloadReq <- struct{}{}
				}
			case err := <-errCh:
				cfg.l.Error(err)
				continue
			}
		}
	}()
}

func (cfg *Config) forEachRoute(do func(alias string, r R.Route, p *PR.Provider)) {
	cfg.proxyProviders.RangeAll(func(_ string, p *PR.Provider) {
		p.RangeRoutes(func(a string, r R.Route) {
			do(a, r, p)
		})
	})
}

func (cfg *Config) load() (res E.NestedError) {
	b := E.NewBuilder("errors loading config")
	defer b.To(&res)

	cfg.l.Debug("loading config")
	defer cfg.l.Debug("loaded config")

	data, err := E.Check(os.ReadFile(common.ConfigPath))
	if err.HasError() {
		b.Add(E.FailWith("read config", err))
		logrus.Fatal(b.Build())
	}

	if !common.NoSchemaValidation {
		if err = Validate(data); err.HasError() {
			b.Add(E.FailWith("schema validation", err))
			logrus.Fatal(b.Build())
		}
	}

	model := types.DefaultConfig()
	if err := E.From(yaml.Unmarshal(data, model)); err.HasError() {
		b.Add(E.FailWith("parse config", err))
		logrus.Fatal(b.Build())
	}

	// errors are non fatal below
	b.Add(cfg.initAutoCert(&model.AutoCert))
	b.Add(cfg.loadProviders(&model.Providers))

	cfg.value = model
	R.SetFindMuxDomains(model.MatchDomains)
	return
}

func (cfg *Config) initAutoCert(autocertCfg *types.AutoCertConfig) (err E.NestedError) {
	if cfg.autocertProvider != nil {
		return
	}

	cfg.l.Debug("initializing autocert")
	defer cfg.l.Debug("initialized autocert")

	cfg.autocertProvider, err = autocert.NewConfig(autocertCfg).GetProvider()
	if err.HasError() {
		err = E.FailWith("autocert provider", err)
	}
	return
}

func (cfg *Config) loadProviders(providers *types.ProxyProviders) (res E.NestedError) {
	cfg.l.Debug("loading providers")
	defer cfg.l.Debug("loaded providers")

	b := E.NewBuilder("errors loading providers")
	defer b.To(&res)

	for _, filename := range providers.Files {
		p, err := PR.NewFileProvider(filename)
		if err != nil {
			b.Add(err.Subject(filename))
			continue
		}
		cfg.proxyProviders.Store(p.GetName(), p)
		b.Add(p.LoadRoutes().Subject(filename))
	}
	for name, dockerHost := range providers.Docker {
		p, err := PR.NewDockerProvider(name, dockerHost)
		if err != nil {
			b.Add(err.Subjectf("%s (%s)", name, dockerHost))
			continue
		}
		cfg.proxyProviders.Store(p.GetName(), p)
		b.Add(p.LoadRoutes().Subject(p.GetName()))
	}
	return
}

func (cfg *Config) controlProviders(action string, do func(*PR.Provider) E.NestedError) {
	errors := E.NewBuilder("errors in %s these providers", action)

	cfg.proxyProviders.RangeAllParallel(func(name string, p *PR.Provider) {
		if err := do(p); err.HasError() {
			errors.Add(err.Subject(p))
		}
	})

	if err := errors.Build(); err.HasError() {
		cfg.l.Error(err)
	}
}

func (cfg *Config) stopProviders() {
	cfg.controlProviders("stop routes", (*PR.Provider).StopAllRoutes)
}
