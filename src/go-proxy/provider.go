package main

import (
	"github.com/sirupsen/logrus"
)

type Provider struct {
	Kind  string `json:"kind"` // docker, file
	Value string `json:"value"`

	watcher     Watcher
	routes      map[string]Route // id -> Route
	l           logrus.FieldLogger
	reloadReqCh chan struct{}
}

// Init is called after LoadProxyConfig
func (p *Provider) Init(name string) error {
	p.l = logrus.WithField("provider", name)
	p.reloadReqCh = make(chan struct{}, 1)

	defer p.initWatcher()

	if err := p.loadProxyConfig(); err != nil {
		return err
	}

	return nil
}

func (p *Provider) StartAllRoutes() {
	ParallelForEachValue(p.routes, Route.Start)
	p.watcher.Start()
}

func (p *Provider) StopAllRoutes() {
	p.watcher.Stop()
	ParallelForEachValue(p.routes, Route.Stop)
	p.routes = nil
}

func (p *Provider) ReloadRoutes() {
	select {
	case p.reloadReqCh <- struct{}{}:
		defer func() {
			<-p.reloadReqCh
		}()

		p.StopAllRoutes()
		err := p.loadProxyConfig()
		if err != nil {
			p.l.Error("failed to reload routes: ", err)
			return
		}
		p.StartAllRoutes()
	default:
		p.l.Info("reload request already in progress")
		return
	}
}

func (p *Provider) loadProxyConfig() error {
	var cfgs ProxyConfigSlice
	var err error

	switch p.Kind {
	case ProviderKind_Docker:
		cfgs, err = p.getDockerProxyConfigs()
	case ProviderKind_File:
		cfgs, err = p.ValidateFile()
	default:
		// this line should never be reached
		return NewNestedError("unknown provider kind")
	}

	if err != nil {
		return err
	}
	p.l.Infof("loaded %d proxy configurations", len(cfgs))

	p.routes = make(map[string]Route, len(cfgs))
	pErrs := NewNestedError("failed to create these routes")

	for _, cfg := range cfgs {
		r, err := NewRoute(&cfg)
		if err != nil {
			pErrs.ExtraError(NewNestedErrorFrom(err).Subject(cfg.Alias))
			continue
		}
		p.routes[cfg.GetID()] = r
	}

	if pErrs.HasExtras() {
		p.routes = nil
		return pErrs
	}
	return nil
}

func (p *Provider) initWatcher() error {
	switch p.Kind {
	case ProviderKind_Docker:
		dockerClient, err := p.getDockerClient()
		if err != nil {
			return NewNestedError("unable to create docker client").With(err)
		}
		p.watcher = p.NewDockerWatcher(dockerClient)
	case ProviderKind_File:
		p.watcher = p.NewFileWatcher()
	}
	return nil
}
