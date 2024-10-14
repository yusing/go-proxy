package v1

import (
	"net/http"

	. "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

func CheckHealth(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("target")
	if target == "" {
		HandleErr(w, r, ErrMissingKey("target"), http.StatusBadRequest)
		return
	}

	status, ok := health.Inspect(target)
	if !ok {
		HandleErr(w, r, ErrNotFound("target", target), http.StatusNotFound)
		return
	}
	WriteBody(w, []byte(status.String()))
}
