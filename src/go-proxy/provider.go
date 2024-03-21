package main

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type Provider struct {
	Kind  string // docker, file
	Value string

	watcher Watcher
	routes  map[string]Route // id -> Route
	mutex   sync.Mutex
	l       logrus.FieldLogger
}

// Init is called after LoadProxyConfig
func (p *Provider) Init(name string) error {
	p.l = prlog.WithFields(logrus.Fields{"kind": p.Kind, "name": name})

	if err := p.loadProxyConfig(); err != nil {
		return err
	}

	p.initWatcher()
	return nil
}

func (p *Provider) StartAllRoutes() {
	ParallelForEachValue(p.routes, Route.Start)
	p.watcher.Start()
}

func (p *Provider) StopAllRoutes() {
	p.watcher.Stop()
	ParallelForEachValue(p.routes, Route.Stop)
	p.routes = make(map[string]Route)
}

func (p *Provider) ReloadRoutes() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.StopAllRoutes()
	err := p.loadProxyConfig()
	if err != nil {
		p.l.Error("failed to reload routes: ", err)
		return
	}
	p.StartAllRoutes()
}

func (p *Provider) loadProxyConfig() error {
	var cfgs []*ProxyConfig
	var err error

	switch p.Kind {
	case ProviderKind_Docker:
		cfgs, err = p.getDockerProxyConfigs()
	case ProviderKind_File:
		cfgs, err = p.getFileProxyConfigs()
	default:
		// this line should never be reached
		return fmt.Errorf("unknown provider kind")
	}

	if err != nil {
		return err
	}
	p.l.Infof("loaded %d proxy configurations", len(cfgs))

	p.routes = make(map[string]Route, len(cfgs))
	for _, cfg := range cfgs {
		r, err := NewRoute(cfg)
		if err != nil {
			p.l.Errorf("error creating route %s: %v", cfg.Alias, err)
			continue
		}
		p.routes[cfg.GetID()] = r
	}

	return nil
}

func (p *Provider) initWatcher() error {
	switch p.Kind {
	case ProviderKind_Docker:
		var err error
		dockerClient, err := p.getDockerClient()
		if err != nil {
			return fmt.Errorf("unable to create docker client: %v", err)
		}
		p.watcher = NewDockerWatcher(dockerClient, p.ReloadRoutes)
	case ProviderKind_File:
		p.watcher = NewFileWatcher(p.Value, p.ReloadRoutes, p.StopAllRoutes)
	}
	return nil
}
