package main

import (
	"os"

	"github.com/yusing/go-proxy/agent/pkg/agent"
	"github.com/yusing/go-proxy/agent/pkg/env"
	"github.com/yusing/go-proxy/agent/pkg/server"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/pkg"
)

func main() {
	args := os.Args
	if len(args) > 1 && args[1] == "migrate" {
		if err := agent.MigrateFromOld(); err != nil {
			E.LogFatal("failed to migrate from old docker compose", err)
		}
		return
	}
	_ = os.Chmod("/app/compose.yml", 0600)
	ca := &agent.PEMPair{}
	err := ca.Load(env.AgentCACert)
	if err != nil {
		E.LogFatal("init CA error", err)
	}
	caCert, err := ca.ToTLSCert()
	if err != nil {
		E.LogFatal("init CA error", err)
	}

	srv := &agent.PEMPair{}
	srv.Load(env.AgentSSLCert)
	if err != nil {
		E.LogFatal("init SSL error", err)
	}
	srvCert, err := srv.ToTLSCert()
	if err != nil {
		E.LogFatal("init SSL error", err)
	}

	logging.Info().Msgf("GoDoxy Agent version %s", pkg.GetVersion())
	logging.Info().Msgf("Agent name: %s", env.AgentName)
	logging.Info().Msgf("Agent port: %d", env.AgentPort)

	logging.Info().Msg(`
Tips:
1. To change the agent name, you can set the AGENT_NAME environment variable.
2. To change the agent port, you can set the AGENT_PORT environment variable.
`)

	t := task.RootTask("agent", false)
	opts := server.Options{
		CACert:     caCert,
		ServerCert: srvCert,
		Port:       env.AgentPort,
	}

	server.StartAgentServer(t, opts)

	task.WaitExit(3)
}
