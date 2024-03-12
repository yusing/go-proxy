package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/golang/glog"
)

type Provider struct {
	Kind  string // docker, file
	Value string

	name         string
	stopWatching chan struct{}
	routes       map[string]Route // id -> Route
	dockerClient *client.Client
	mutex        sync.Mutex
	lastUpdate   time.Time
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

func (p *Provider) needUpdate() bool {
	return p.lastUpdate.Add(1 * time.Second).Before(time.Now())
}

func (p *Provider) StopAllRoutes() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.needUpdate() {
		return
	}

	close(p.stopWatching)
	if p.dockerClient != nil {
		p.dockerClient.Close()
	}

	var wg sync.WaitGroup
	wg.Add(len(p.routes))

	for _, route := range p.routes {
		go func(r Route) {
			r.StopListening()
			r.RemoveFromRoutes()
			wg.Done()
		}(route)
	}
	wg.Wait()
	p.routes = make(map[string]Route)
}

func (p *Provider) BuildStartRoutes() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.needUpdate() {
		return
	}

	p.lastUpdate = time.Now()
	p.stopWatching = make(chan struct{})
	p.routes = make(map[string]Route)

	cfgs, err := p.GetProxyConfigs()
	if err != nil {
		p.Logf("Build", "unable to get proxy configs: %v", err)
		return
	}

	for _, cfg := range cfgs {
		r, err := NewRoute(cfg)
		if err != nil {
			p.Logf("Build", "error creating route %q: %v", cfg.Alias, err)
			continue
		}
		r.SetupListen()
		r.Listen()
		p.routes[cfg.GetID()] = r
	}
	p.WatchChanges()
	p.Logf("Build", "built %d routes", len(p.routes))
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

func (p *Provider) Logf(t string, s string, args ...interface{}) {
	glog.Infof("[%s] %s provider %q: "+s, append([]interface{}{t, p.Kind, p.name}, args...)...)
}

func (p *Provider) Errorf(t string, s string, args ...interface{}) {
	glog.Errorf("[%s] %s provider %q: "+s, append([]interface{}{t, p.Kind, p.name}, args...)...)
}

func (p *Provider) Warningf(t string, s string, args ...interface{}) {
	glog.Warningf("[%s] %s provider %q: "+s, append([]interface{}{t, p.Kind, p.name}, args...)...)
}
