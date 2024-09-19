package v1

import (
	"net/http"

	U "github.com/yusing/go-proxy/api/v1/utils"
	"github.com/yusing/go-proxy/config"
)

func Reload(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	if err := cfg.Reload().Error(); err != nil {
		U.HandleErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
