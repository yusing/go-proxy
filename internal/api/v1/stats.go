package v1

import (
	"context"
	"net/http"
	"time"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/server"
	"github.com/yusing/go-proxy/internal/utils"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func Stats(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	U.HandleErr(w, r, U.RespondJson(w, getStats(cfg)))
}

func StatsWS(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	localAddresses := []string{"127.0.0.1", "10.0.*.*", "172.16.*.*", "192.168.*.*"}
	originPats := make([]string, len(cfg.Value().MatchDomains)+len(localAddresses))

	if len(originPats) == 0 {
		U.Logger.Warnf("no match domains configured, accepting websocket request from all origins")
		originPats = []string{"*"}
	} else {
		for i, domain := range cfg.Value().MatchDomains {
			originPats[i] = "*." + domain
		}
		originPats = append(originPats, localAddresses...)
	}
	if common.IsDebug {
		originPats = []string{"*"}
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: originPats,
	})
	if err != nil {
		U.Logger.Errorf("/stats/ws failed to upgrade websocket: %s", err)
		return
	}
	defer conn.CloseNow()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := getStats(cfg)
		if err := wsjson.Write(ctx, conn, stats); err != nil {
			U.Logger.Errorf("/stats/ws failed to write JSON: %s", err)
			return
		}
	}
}

func getStats(cfg *config.Config) map[string]any {
	return map[string]any{
		"proxies": cfg.Statistics(),
		"uptime":  utils.FormatDuration(server.GetProxyServer().Uptime()),
	}
}
