package v1

import (
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
)

func AgentsWS(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	U.PeriodicWS(cfg.Value().MatchDomains, w, r, 10*time.Second, func(conn *websocket.Conn) error {
		wsjson.Write(r.Context(), conn, cfg.ListAgents())
		return nil
	})
}
