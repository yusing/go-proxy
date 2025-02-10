package provider

import (
	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/agent/pkg/agent"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/watcher"
)

type AgentProvider struct {
	*agent.AgentConfig
	docker ProviderImpl
}

func (p *AgentProvider) ShortName() string {
	return p.Name()
}

func (p *AgentProvider) NewWatcher() watcher.Watcher {
	return p.docker.NewWatcher()
}

func (p *AgentProvider) IsExplicitOnly() bool {
	return p.docker.IsExplicitOnly()
}

func (p *AgentProvider) loadRoutesImpl() (route.Routes, E.Error) {
	return p.docker.loadRoutesImpl()
}

func (p *AgentProvider) Logger() *zerolog.Logger {
	return p.docker.Logger()
}
