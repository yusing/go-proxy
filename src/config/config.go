package config

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/autocert"
	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	PR "github.com/yusing/go-proxy/proxy/provider"
	R "github.com/yusing/go-proxy/route"
	U "github.com/yusing/go-proxy/utils"
	F "github.com/yusing/go-proxy/utils/functional"
	W "github.com/yusing/go-proxy/watcher"
	"github.com/yusing/go-proxy/watcher/events"
	"gopkg.in/yaml.v3"
)

type Config struct {
	value            *M.Config
	proxyProviders   F.Map[string, *PR.Provider]
	autocertProvider *autocert.Provider

	l logrus.FieldLogger

	watcher       W.Watcher
	watcherCtx    context.Context
	watcherCancel context.CancelFunc
	reloadReq     chan struct{}
}

func Load() (*Config, E.NestedError) {
	cfg := &Config{
		proxyProviders: F.NewMapOf[string, *PR.Provider](),
		l:              logrus.WithField("module", "config"),
		watcher:        W.NewFileWatcher(common.ConfigFileName),
		reloadReq:      make(chan struct{}, 1),
	}
	return cfg, cfg.load()
}

func Validate(data []byte) E.NestedError {
	return U.ValidateYaml(U.GetSchema(common.ConfigSchemaPath), data)
}

func (cfg *Config) Value() M.Config {
	return *cfg.value
}

func (cfg *Config) GetAutoCertProvider() *autocert.Provider {
	return cfg.autocertProvider
}

func (cfg *Config) Dispose() {
	if cfg.watcherCancel != nil {
		cfg.watcherCancel()
		cfg.l.Debug("stopped watcher")
	}
	cfg.stopProviders()
}

func (cfg *Config) Reload() E.NestedError {
	cfg.stopProviders()
	if err := cfg.load(); err.HasError() {
		return err
	}
	cfg.StartProxyProviders()
	return nil
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
				if event.Action == events.ActionFileDeleted {
					cfg.stopProviders()
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
		return
	}

	if !common.NoSchemaValidation {
		if err = Validate(data); err.HasError() {
			b.Add(E.FailWith("schema validation", err))
			return
		}
	}

	model := M.DefaultConfig()
	if err := E.From(yaml.Unmarshal(data, model)); err.HasError() {
		b.Add(E.FailWith("parse config", err))
		return
	}

	// errors are non fatal below
	b.WithSeverity(E.SeverityWarning)
	b.Add(cfg.initAutoCert(&model.AutoCert))
	b.Add(cfg.loadProviders(&model.Providers))

	cfg.value = model
	return
}

func (cfg *Config) initAutoCert(autocertCfg *M.AutoCertConfig) (err E.NestedError) {
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

func (cfg *Config) loadProviders(providers *M.ProxyProviders) (res E.NestedError) {
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
		b.Add(p.LoadRoutes().Subject(dockerHost))
	}
	return
}

func (cfg *Config) controlProviders(action string, do func(*PR.Provider) E.NestedError) {
	errors := E.NewBuilder("errors in %s these providers", action)

	cfg.proxyProviders.RangeAll(func(name string, p *PR.Provider) {
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
