package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/server"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func Stats(w http.ResponseWriter, r *http.Request) {
	U.RespondJSON(w, r, getStats())
}

func StatsWS(w http.ResponseWriter, r *http.Request) {
	var originPats []string

	localAddresses := []string{"127.0.0.1", "10.0.*.*", "172.16.*.*", "192.168.*.*"}

	if len(config.Value().MatchDomains) == 0 {
		U.LogWarn(r).Msg("no match domains configured, accepting websocket API request from all origins")
		originPats = []string{"*"}
	} else {
		originPats = make([]string, len(config.Value().MatchDomains))
		for i, domain := range config.Value().MatchDomains {
			originPats[i] = "*" + domain
		}
		originPats = append(originPats, localAddresses...)
	}
	U.LogInfo(r).Msgf("websocket API request from origins: %s", originPats)
	if common.IsDebug {
		originPats = []string{"*"}
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: originPats,
	})
	if err != nil {
		U.LogError(r).Err(err).Msg("failed to upgrade websocket")
		return
	}
	/* trunk-ignore(golangci-lint/errcheck) */
	defer conn.CloseNow()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := getStats()
		if err := wsjson.Write(ctx, conn, stats); err != nil {
			U.LogError(r).Msg("failed to write JSON")
			return
		}
	}
}

func getStats() map[string]any {
	return map[string]any{
		"proxies": config.Statistics(),
		"uptime":  strutils.FormatDuration(server.GetProxyServer().Uptime()),
	}
}
