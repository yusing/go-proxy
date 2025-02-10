package handler

import (
	"net/http"

	"github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/metrics"
)

func SystemInfo(w http.ResponseWriter, r *http.Request) {
	info, err := metrics.GetSystemInfo(r.Context())
	if err != nil {
		utils.HandleErr(w, r, err)
		return
	}
	utils.RespondJSON(w, r, info)
}
