package v1

import (
	"net/http"

	"github.com/coder/websocket/wsjson"
	agentPkg "github.com/yusing/go-proxy/agent/pkg/agent"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/metrics/systeminfo"
	"github.com/yusing/go-proxy/internal/net/http/httpheaders"
)

func SystemInfo(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	agentName := query.Get("agent_name")
	query.Del("agent_name")
	if agentName == "" {
		systeminfo.Poller.ServeHTTP(w, r)
		return
	}

	agent, ok := cfg.GetAgent(agentName)
	if !ok {
		U.HandleErr(w, r, U.ErrInvalidKey("agent_name"), http.StatusNotFound)
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
		r = r.WithContext(r.Context())
		clientConn, err := U.InitiateWS(w, r)
		if err != nil {
			U.HandleErr(w, r, E.Wrap(err, "failed to initiate websocket"))
			return
		}
		defer clientConn.CloseNow()
		agentConn, _, err := agent.Websocket(r.Context(), agentPkg.EndpointSystemInfo+"?"+query.Encode())
		if err != nil {
			U.HandleErr(w, r, E.Wrap(err, "failed to connect to agent with websocket"))
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
					U.HandleErr(w, r, E.Wrap(err, "failed to write data to client"))
					return
				}
			}
		}
	}
}
