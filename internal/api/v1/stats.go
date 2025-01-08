package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func Stats(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	U.RespondJSON(w, r, getStats(cfg))
}

func StatsWS(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	var originPats []string

	localAddresses := []string{"127.0.0.1", "10.0.*.*", "172.16.*.*", "192.168.*.*"}

	if len(cfg.Value().MatchDomains) == 0 {
		U.LogWarn(r).Msg("no match domains configured, accepting websocket API request from all origins")
		originPats = []string{"*"}
	} else {
		originPats = make([]string, len(cfg.Value().MatchDomains))
		for i, domain := range cfg.Value().MatchDomains {
			originPats[i] = "*" + domain
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
		stats := getStats(cfg)
		if err := wsjson.Write(ctx, conn, stats); err != nil {
			U.LogError(r).Msg("failed to write JSON")
			return
		}
	}
}

var startTime = time.Now()

func getStats(cfg config.ConfigInstance) map[string]any {
	return map[string]any{
		"proxies": cfg.Statistics(),
		"uptime":  strutils.FormatDuration(time.Since(startTime)),
	}
}
