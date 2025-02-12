package utils

import (
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/net/http/httpheaders"
)

func warnNoMatchDomains() {
	logging.Warn().Msg("no match domains configured, accepting websocket API request from all origins")
}

var warnNoMatchDomainOnce sync.Once

func InitiateWS(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	var originPats []string

	localAddresses := []string{"127.0.0.1", "10.0.*.*", "172.16.*.*", "192.168.*.*"}

	allowedDomains := httpheaders.WebsocketAllowedDomains(r.Header)
	if len(allowedDomains) == 0 || common.IsDebug {
		warnNoMatchDomainOnce.Do(warnNoMatchDomains)
		originPats = []string{"*"}
	} else {
		originPats = make([]string, len(allowedDomains))
		for i, domain := range allowedDomains {
			if domain[0] != '.' {
				originPats[i] = "*." + domain
			} else {
				originPats[i] = "*" + domain
			}
		}
		originPats = append(originPats, localAddresses...)
	}
	return websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: originPats,
	})
}

func PeriodicWS(w http.ResponseWriter, r *http.Request, interval time.Duration, do func(conn *websocket.Conn) error) {
	conn, err := InitiateWS(w, r)
	if err != nil {
		HandleErr(w, r, err)
		return
	}
	//nolint:errcheck
	defer conn.CloseNow()

	if err := do(conn); err != nil {
		HandleErr(w, r, err)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
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
