package config

import (
	"github.com/yusing/go-proxy/agent/pkg/agent"
	"github.com/yusing/go-proxy/internal/gperr"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/route/provider"
	"github.com/yusing/go-proxy/internal/utils/functional"
)

var agentPool = functional.NewMapOf[string, *agent.AgentConfig]()

func addAgent(agent *agent.AgentConfig) {
	agentPool.Store(agent.Addr, agent)
}

func removeAllAgents() {
	agentPool.Clear()
}

func GetAgent(addr string) (agent *agent.AgentConfig, ok bool) {
	agent, ok = agentPool.Load(addr)
	return
}

func (cfg *Config) GetAgent(agentAddrOrDockerHost string) (*agent.AgentConfig, bool) {
	if !agent.IsDockerHostAgent(agentAddrOrDockerHost) {
		return GetAgent(agentAddrOrDockerHost)
	}
	return GetAgent(agent.GetAgentAddrFromDockerHost(agentAddrOrDockerHost))
}

func (cfg *Config) AddAgent(host string, ca agent.PEMPair, client agent.PEMPair) (int, gperr.Error) {
	var agentCfg agent.AgentConfig
	agentCfg.Addr = host
	err := agentCfg.StartWithCerts(cfg.Task(), ca.Cert, client.Cert, client.Key)
	if err != nil {
		return 0, gperr.Wrap(err, "failed to start agent")
	}
	addAgent(&agentCfg)

	provider := provider.NewAgentProvider(&agentCfg)
	if err := cfg.errIfExists(provider); err != nil {
		return 0, err
	}
	provider.LoadRoutes()
	provider.Start(cfg.Task())
	cfg.storeProvider(provider)
	logging.Info().Msgf("Added agent %s with %d routes", host, provider.NumRoutes())
	return provider.NumRoutes(), nil
}

func (cfg *Config) ListAgents() []*agent.AgentConfig {
	agents := make([]*agent.AgentConfig, 0, agentPool.Size())
	agentPool.RangeAll(func(key string, value *agent.AgentConfig) {
		agents = append(agents, value)
	})
	return agents
}
