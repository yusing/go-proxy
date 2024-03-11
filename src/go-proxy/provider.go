package main

import (
	"fmt"
	"sync"

	"github.com/docker/docker/client"
	"github.com/golang/glog"
)

type Provider struct {
	Kind  string // docker, file
	Value string

	name         string
	stopWatching chan struct{}
	routes       SafeMap[string, Route] // id -> Route
	dockerClient *client.Client
}

func (p *Provider) GetProxyConfigs() ([]*ProxyConfig, error) {
	switch p.Kind {
	case ProviderKind_Docker:
		return p.getDockerProxyConfigs()
	case ProviderKind_File:
		return p.getFileProxyConfigs()
	default:
		// this line should never be reached
		return nil, fmt.Errorf("unknown provider kind %q", p.Kind)
	}
}

func (p *Provider) StopAllRoutes() {
	close(p.stopWatching)
	if p.dockerClient != nil {
		p.dockerClient.Close()
	}

	var wg sync.WaitGroup
	wg.Add(p.routes.Size())

	for _, route := range p.routes.Iterator() {
		go func(r Route) {
			r.StopListening()
			r.RemoveFromRoutes()
			wg.Done()
		}(route)
	}
	wg.Wait()
	p.routes = NewSafeMap[string, Route]()
}

func (p *Provider) BuildStartRoutes() {
	p.stopWatching = make(chan struct{})
	p.routes = NewSafeMap[string, Route]()

	cfgs, err := p.GetProxyConfigs()
	if err != nil {
		p.Logf("Build", "unable to get proxy configs: %v", p.name, err)
		return
	}

	for _, cfg := range cfgs {
		r, err := NewRoute(cfg)
		if err != nil {
			p.Logf("Build", "error creating route %q: %v", p.name, cfg.Alias, err)
			continue
		}
		r.SetupListen()
		r.Listen()
		p.routes.Set(cfg.GetID(), r)
	}
	p.WatchChanges()
	p.Logf("Build", "built %d routes", p.routes.Size())
}

func (p *Provider) WatchChanges() {
	switch p.Kind {
	case ProviderKind_Docker:
		go p.grWatchDockerChanges()
	case ProviderKind_File:
		go p.grWatchFileChanges()
	default:
		// this line should never be reached
		p.Errorf("unknown provider kind %q", p.Kind)
	}
}

func (p* Provider) Logf(t string, s string, args ...interface{}) {
	glog.Infof("[%s] %s provider %q: " + s, append([]interface{}{t, p.Kind, p.name}, args...)...)
}

func (p* Provider) Errorf(t string, s string, args ...interface{}) {
	glog.Errorf("[%s] %s provider %q: " + s, append([]interface{}{t, p.Kind, p.name}, args...)...)
}

func (p* Provider) Warningf(t string, s string, args ...interface{}) {
	glog.Warningf("[%s] %s provider %q: " + s, append([]interface{}{t, p.Kind, p.name}, args...)...)
}