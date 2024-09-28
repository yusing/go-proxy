package v1

import (
	"net/http"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/server"
	"github.com/yusing/go-proxy/internal/utils"
)

func Stats(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	stats := map[string]any{
		"proxies": cfg.Statistics(),
		"uptime":  utils.FormatDuration(server.GetProxyServer().Uptime()),
	}
	if err := U.RespondJson(w, stats); err != nil {
		U.HandleErr(w, r, err)
	}
}
