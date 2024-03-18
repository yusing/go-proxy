package main

import (
	"fmt"
	"sync"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type Provider struct {
	Kind  string // docker, file
	Value string

	name         string
	watcher      Watcher
	routes       map[string]Route // id -> Route
	dockerClient *client.Client
	mutex        sync.Mutex
	l logrus.FieldLogger
}

func (p *Provider) Setup() error {
	var cfgs []*ProxyConfig
	var err error

	p.l = prlog.WithFields(logrus.Fields{"kind": p.Kind, "name": p.name})

	switch p.Kind {
	case ProviderKind_Docker:
		cfgs, err = p.getDockerProxyConfigs()
		p.watcher = NewDockerWatcher(p.dockerClient, p.ReloadRoutes)
	case ProviderKind_File:
		cfgs, err = p.getFileProxyConfigs()
		p.watcher = NewFileWatcher(p.Value, p.ReloadRoutes, p.StopAllRoutes)
	default:
		// this line should never be reached
		return fmt.Errorf("unknown provider kind")
	}

	if err != nil {
		return err
	}
	p.l.Infof("loaded %d proxy configurations", len(cfgs))

	for _, cfg := range cfgs {
		r, err := NewRoute(cfg)
		if err != nil {
			p.l.Errorf("error creating route %s: %v", cfg.Alias, err)
			continue
		}
		r.SetupListen()
		r.Listen()
		p.routes[cfg.GetID()] = r
	}
	return nil
}

func (p *Provider) StartAllRoutes() {
	p.routes = make(map[string]Route)
	err := p.Setup()
	if err != nil {
		p.l.Error(err)
		return
	}
	p.watcher.Start()
}

func (p *Provider) StopAllRoutes() {
	p.watcher.Stop()
	p.dockerClient = nil

	ParallelForEachValue(p.routes, func(r Route) {
		r.StopListening()
		r.RemoveFromRoutes()
	})

	p.routes = make(map[string]Route)
}

func (p *Provider) ReloadRoutes() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.StopAllRoutes()
	p.StartAllRoutes()
}