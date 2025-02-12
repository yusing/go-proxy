package v1

import (
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/route/routes/routequery"
)

func HealthWS(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	U.PeriodicWS(cfg.Value().MatchDomains, w, r, 1*time.Second, func(conn *websocket.Conn) error {
		return wsjson.Write(r.Context(), conn, routequery.HealthMap())
	})
}
