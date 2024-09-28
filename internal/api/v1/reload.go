package v1

import (
	"net/http"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/config"
)

func Reload(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	if err := cfg.Reload(); err != nil {
		U.RespondJson(w, err.JSONObject(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
