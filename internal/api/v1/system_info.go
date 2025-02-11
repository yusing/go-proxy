package v1

import (
	"net/http"

	agentPkg "github.com/yusing/go-proxy/agent/pkg/agent"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/metrics"
)

func SystemInfo(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	agentName := r.FormValue("agent_name")
	if agentName == "" {
		info, err := metrics.GetSystemInfo(r.Context())
		if err != nil {
			U.HandleErr(w, r, err)
			return
		}
		U.RespondJSON(w, r, info)
	} else {
		agent, ok := cfg.GetAgent(agentName)
		if !ok {
			U.HandleErr(w, r, U.ErrInvalidKey("agent_name"), http.StatusNotFound)
			return
		}
		respData, status, err := agent.Fetch(r.Context(), agentPkg.EndpointSystemInfo)
		if err != nil {
			U.HandleErr(w, r, err)
			return
		}
		if status != http.StatusOK {
			http.Error(w, string(respData), status)
			return
		}
		U.RespondJSON(w, r, respData)
	}
}
