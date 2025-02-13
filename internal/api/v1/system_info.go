package v1

import (
	"net/http"

	agentPkg "github.com/yusing/go-proxy/agent/pkg/agent"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/metrics/systeminfo"
	"github.com/yusing/go-proxy/internal/net/http/httpheaders"
	"github.com/yusing/go-proxy/internal/net/http/reverseproxy"
)

func SystemInfo(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	agentAddr := query.Get("agent_addr")
	query.Del("agent_addr")
	if agentAddr == "" {
		systeminfo.Poller.ServeHTTP(w, r)
		return
	}

	agent, ok := cfg.GetAgent(agentAddr)
	if !ok {
		U.HandleErr(w, r, U.ErrInvalidKey("agent_addr"), http.StatusNotFound)
		return
	}

	isWS := httpheaders.IsWebsocket(r.Header)
	if !isWS {
		respData, status, err := agent.Forward(r, agentPkg.EndpointSystemInfo)
		if err != nil {
			U.HandleErr(w, r, E.Wrap(err, "failed to forward request to agent"))
			return
		}
		if status != http.StatusOK {
			http.Error(w, string(respData), status)
			return
		}
		U.WriteBody(w, respData)
	} else {
		rp := reverseproxy.NewReverseProxy("agent", agentPkg.AgentURL, agent.Transport())
		header := r.Header.Clone()
		r, err := http.NewRequestWithContext(r.Context(), r.Method, agentPkg.EndpointSystemInfo+"?"+query.Encode(), nil)
		if err != nil {
			U.HandleErr(w, r, E.Wrap(err, "failed to create request"))
			return
		}
		r.Header = header
		rp.ServeHTTP(w, r)
	}
}
