package v1

import (
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func Stats(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	U.RespondJSON(w, r, getStats(cfg))
}

func StatsWS(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	U.PeriodicWS(cfg, w, r, 1*time.Second, func(conn *websocket.Conn) error {
		return wsjson.Write(r.Context(), conn, getStats(cfg))
	})
}

var startTime = time.Now()

func getStats(cfg config.ConfigInstance) map[string]any {
	return map[string]any{
		"proxies": cfg.Statistics(),
		"uptime":  strutils.FormatDuration(time.Since(startTime)),
	}
}
