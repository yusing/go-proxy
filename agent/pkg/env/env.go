package env

import (
	"os"

	"github.com/yusing/go-proxy/internal/common"
)

func DefaultAgentName() string {
	name, err := os.Hostname()
	if err != nil {
		return "agent"
	}
	return name
}

var (
	AgentName = common.GetEnvString("AGENT_NAME", DefaultAgentName())
	AgentPort = common.GetEnvInt("AGENT_PORT", 8890)
)
