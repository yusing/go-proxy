package v1

import (
	"net/http"
	"strings"

	"github.com/coder/websocket/wsjson"
	agentPkg "github.com/yusing/go-proxy/agent/pkg/agent"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/metrics/systeminfo"
)

func SystemInfo(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	isWS := strings.HasSuffix(r.URL.Path, "/ws")
	agentName := r.URL.Query().Get("agent_name")
	if agentName == "" {
		if isWS {
			systeminfo.Poller.ServeWS(cfg, w, r)
		} else {
			systeminfo.Poller.ServeHTTP(w, r)
		}
		return
	}

	agent, ok := cfg.GetAgent(agentName)
	if !ok {
		U.HandleErr(w, r, U.ErrInvalidKey("agent_name"), http.StatusNotFound)
		return
	}
	if !isWS {
		respData, status, err := agent.Forward(r, agentPkg.EndpointSystemInfo)
		if err != nil {
			U.HandleErr(w, r, err)
			return
		}
		if status != http.StatusOK {
			http.Error(w, string(respData), status)
			return
		}
		U.WriteBody(w, respData)
	} else {
		clientConn, err := U.InitiateWS(cfg, w, r)
		if err != nil {
			U.HandleErr(w, r, err)
			return
		}
		agentConn, _, err := agent.Websocket(r.Context(), agentPkg.EndpointSystemInfo)
		if err != nil {
			U.HandleErr(w, r, err)
			return
		}
		//nolint:errcheck
		defer agentConn.CloseNow()
		var data []byte
		for {
			select {
			case <-r.Context().Done():
				return
			default:
				err := wsjson.Read(r.Context(), agentConn, &data)
				if err == nil {
					err = wsjson.Write(r.Context(), clientConn, data)
				}
				if err != nil {
					U.HandleErr(w, r, err)
					return
				}
			}
		}
	}
}
