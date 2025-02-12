package v1

import (
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/net/http/httpheaders"
	"github.com/yusing/go-proxy/internal/route/routes/routequery"
)

func Health(w http.ResponseWriter, r *http.Request) {
	if httpheaders.IsWebsocket(r.Header) {
		U.PeriodicWS(w, r, 1*time.Second, func(conn *websocket.Conn) error {
			return wsjson.Write(r.Context(), conn, routequery.HealthMap())
		})
	} else {
		U.RespondJSON(w, r, routequery.HealthMap())
	}
}
