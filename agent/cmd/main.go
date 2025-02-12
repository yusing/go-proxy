package main

import (
	"fmt"

	"github.com/yusing/go-proxy/agent/pkg/agent"
	"github.com/yusing/go-proxy/agent/pkg/certs"
	"github.com/yusing/go-proxy/agent/pkg/env"
	"github.com/yusing/go-proxy/agent/pkg/server"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/pkg"
	"gopkg.in/yaml.v3"
)

func printNewClientHelp() {
	ip, ok := agent.MachineIP()
	if !ok {
		logging.Warn().Msg("No valid network interface found, change <machine-ip> to your actual IP")
		ip = "<machine-ip>"
	} else {
		logging.Info().Msgf("Detected machine IP: %s, change if needed", ip)
	}

	host := fmt.Sprintf("%s:%d", ip, env.AgentPort)
	cfgYAML, _ := yaml.Marshal(map[string]any{
		"providers": map[string]any{
			"agents": host,
		},
	})

	logging.Info().Msgf("On main server, run:\n\ndocker exec godoxy /app/run add-agent '%s'\n", host)
	logging.Info().Msgf("Then add this host (%s) to main server config like below:\n", host)
	logging.Info().Msg(string(cfgYAML))
}

func main() {
	ca, srv, isNew, err := certs.InitCerts()
	if err != nil {
		E.LogFatal("init CA error", err)
	}

	logging.Info().Msgf("GoDoxy Agent version %s", pkg.GetVersion())
	logging.Info().Msgf("Agent name: %s", env.AgentName)
	logging.Info().Msgf("Agent port: %d", env.AgentPort)

	logging.Info().Msg(`
Tips:
1. To change the agent name, you can set the AGENT_NAME environment variable.
2. To change the agent port, you can set the AGENT_PORT environment variable.
3. To skip the version check, you can set AGENT_SKIP_VERSION_CHECK to true.
4. If anything goes wrong, you can remove the 'certs' directory and start over.
`)

	t := task.RootTask("agent", false)
	opts := server.Options{
		CACert:     ca,
		ServerCert: srv,
		Port:       env.AgentPort,
	}

	if isNew {
		logging.Info().Msg("Initialization complete.")
		printNewClientHelp()
		server.StartRegistrationServer(t, opts)
	}

	server.StartAgentServer(t, opts)

	task.WaitExit(3)
}
