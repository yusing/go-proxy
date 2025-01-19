package utils

import (
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/yusing/go-proxy/internal/common"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/logging"
)

func warnNoMatchDomains() {
	logging.Warn().Msg("no match domains configured, accepting websocket API request from all origins")
}

var warnNoMatchDomainOnce sync.Once

func InitiateWS(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	var originPats []string

	localAddresses := []string{"127.0.0.1", "10.0.*.*", "172.16.*.*", "192.168.*.*"}

	if len(cfg.Value().MatchDomains) == 0 {
		warnNoMatchDomainOnce.Do(warnNoMatchDomains)
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
	return websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: originPats,
	})
}

func PeriodicWS(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request, interval time.Duration, do func(conn *websocket.Conn) error) {
	conn, err := InitiateWS(cfg, w, r)
	if err != nil {
		HandleErr(w, r, err)
		return
	}
	/* trunk-ignore(golangci-lint/errcheck) */
	defer conn.CloseNow()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-cfg.Context().Done():
			return
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := do(conn); err != nil {
				HandleErr(w, r, err)
				return
			}
		}
	}
}
