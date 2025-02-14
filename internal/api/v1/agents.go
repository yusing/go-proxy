package v1

import (
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/net/http/httpheaders"
)

func ListAgents(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	if httpheaders.IsWebsocket(r.Header) {
		U.PeriodicWS(w, r, 10*time.Second, func(conn *websocket.Conn) error {
			wsjson.Write(r.Context(), conn, cfg.ListAgents())
			return nil
		})
	} else {
		U.RespondJSON(w, r, cfg.ListAgents())
	}
}
