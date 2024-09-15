package config

import (
	"context"

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
	"gopkg.in/yaml.v3"
)

type Config struct {
	value *M.Config

	l                logrus.FieldLogger
	reader           U.Reader
	proxyProviders   *F.Map[string, *PR.Provider]
	autocertProvider *autocert.Provider

	watcher       W.Watcher
	watcherCtx    context.Context
	watcherCancel context.CancelFunc
	reloadReq     chan struct{}
}

func New() (*Config, E.NestedError) {
	cfg := &Config{
		l:         logrus.WithField("module", "config"),
		reader:    U.NewFileReader(common.ConfigPath),
		watcher:   W.NewFileWatcher(common.ConfigFileName),
		reloadReq: make(chan struct{}, 1),
	}
	if err := cfg.load(); err.IsNotNil() {
		return nil, err
	}
	cfg.startProviders()
	cfg.watchChanges()
	return cfg, E.Nil()
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
	cfg.watcherCancel()
	cfg.l.Debug("stopped watcher")
	cfg.stopProviders()
	cfg.l.Debug("stopped providers")
}

func (cfg *Config) Reload() E.NestedError {
	cfg.stopProviders()
	if err := cfg.load(); err.IsNotNil() {
		return err
	}
	cfg.startProviders()
	return E.Nil()
}

func (cfg *Config) FindRoute(alias string) R.Route {
	r := cfg.proxyProviders.Find(
		func(p *PR.Provider) (any, bool) {
			rs := p.GetCurrentRoutes()
			if rs.Contains(alias) {
				return rs.Get(alias), true
			}
			return nil, false
		},
	)
	if r == nil {
		return nil
	}
	return r.(R.Route)
}

func (cfg *Config) RoutesByAlias() map[string]U.SerializedObject {
	routes := make(map[string]U.SerializedObject)
	cfg.proxyProviders.Each(func(p *PR.Provider) {
		prName := p.GetName()
		p.GetCurrentRoutes().EachKV(func(a string, r R.Route) {
			obj, err := U.Serialize(r)
			if err != nil {
				cfg.l.Error(err)
				return
			}
			obj["provider"] = prName
			switch r.(type) {
			case *R.StreamRoute:
				obj["type"] = "stream"
			case *R.HTTPRoute:
				obj["type"] = "reverse_proxy"
			default:
				panic("bug: should not reach here")
			}
			routes[a] = obj
		})
	})
	return routes
}

func (cfg *Config) Statistics() map[string]interface{} {
	nTotalStreams := 0
	nTotalRPs := 0
	providerStats := make(map[string]interface{})

	cfg.proxyProviders.Each(func(p *PR.Provider) {
		stats := make(map[string]interface{})
		nStreams := 0
		nRPs := 0
		p.GetCurrentRoutes().EachKV(func(a string, r R.Route) {
			switch r.(type) {
			case *R.StreamRoute:
				nStreams++
				nTotalStreams++
			case *R.HTTPRoute:
				nRPs++
				nTotalRPs++
			default:
				panic("bug: should not reach here")
			}
		})
		stats["type"] = p.GetType()
		stats["num_streams"] = nStreams
		stats["num_reverse_proxies"] = nRPs
		providerStats[p.GetName()] = stats
	})

	return map[string]interface{}{
		"num_total_streams":         nTotalStreams,
		"num_total_reverse_proxies": nTotalRPs,
		"providers":                 providerStats,
	}
}

func (cfg *Config) watchChanges() {
	cfg.watcherCtx, cfg.watcherCancel = context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-cfg.watcherCtx.Done():
				return
			case <-cfg.reloadReq:
				if err := cfg.Reload(); err.IsNotNil() {
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
				if event.Action.IsDelete() {
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

func (cfg *Config) load() E.NestedError {
	cfg.l.Debug("loading config")

	data, err := cfg.reader.Read()
	if err.IsNotNil() {
		return E.Failure("read config").With(err)
	}

	model := M.DefaultConfig()
	if err := E.From(yaml.Unmarshal(data, model)); err.IsNotNil() {
		return E.Failure("parse config").With(err)
	}

	if !common.NoSchemaValidation {
		if err = Validate(data); err.IsNotNil() {
			return err
		}
	}

	warnings := E.NewBuilder("errors loading config")

	cfg.l.Debug("starting autocert")
	ap, err := autocert.NewConfig(&model.AutoCert).GetProvider()
	if err.IsNotNil() {
		warnings.Add(E.Failure("autocert provider").With(err))
	} else {
		cfg.l.Debug("started autocert")
	}
	cfg.autocertProvider = ap

	cfg.l.Debug("loading providers")
	cfg.proxyProviders = F.NewMap[string, *PR.Provider]()
	for _, filename := range model.Providers.Files {
		p := PR.NewFileProvider(filename)
		cfg.proxyProviders.Set(p.GetName(), p)
	}
	for name, dockerHost := range model.Providers.Docker {
		p := PR.NewDockerProvider(name, dockerHost)
		cfg.proxyProviders.Set(p.GetName(), p)
	}
	cfg.l.Debug("loaded providers")

	cfg.value = model

	if err := warnings.Build(); err.IsNotNil() {
		cfg.l.Warn(err)
	}

	cfg.l.Debug("loaded config")
	return E.Nil()
}

func (cfg *Config) controlProviders(action string, do func(*PR.Provider) E.NestedError) {
	errors := E.NewBuilder("cannot %s these providers", action)

	cfg.proxyProviders.EachKVParallel(func(name string, p *PR.Provider) {
		if err := do(p); err.IsNotNil() {
			errors.Add(E.From(err).Subject(p))
		}
	})

	if err := errors.Build(); err.IsNotNil() {
		cfg.l.Error(err)
	}
}

func (cfg *Config) startProviders() {
	cfg.controlProviders("start", (*PR.Provider).StartAllRoutes)
}

func (cfg *Config) stopProviders() {
	cfg.controlProviders("stop routes", (*PR.Provider).StopAllRoutes)
}
