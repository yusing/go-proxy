package v1

import (
	"net/http"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	config "github.com/yusing/go-proxy/internal/config/types"
)

func Reload(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request) {
	if err := cfg.Reload(); err != nil {
		U.HandleErr(w, r, err)
		return
	}
	U.WriteBody(w, []byte("OK"))
}
