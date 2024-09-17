package v1

import (
	"net/http"

	U "github.com/yusing/go-proxy/api/v1/utils"
	"github.com/yusing/go-proxy/config"
	"github.com/yusing/go-proxy/server"
	"github.com/yusing/go-proxy/utils"
)

func Stats(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	stats := map[string]any{
		"proxies": cfg.Statistics(),
		"uptime":  utils.FormatDuration(server.GetProxyServer().Uptime()),
	}
	if err := U.RespondJson(stats, w); err != nil {
		U.HandleErr(w, r, err)
	}
}
