package env

import "github.com/yusing/go-proxy/internal/common"

var (
	AgentName = common.GetEnvString("AGENT_NAME", "agent")
	AgentPort = common.GetEnvInt("AGENT_PORT", 8890)
)
