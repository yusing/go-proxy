package v1

import (
	"net/http"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/config"
)

func Reload(w http.ResponseWriter, r *http.Request) {
	if err := config.Reload(); err != nil {
		U.RespondJSON(w, r, err.JSONObject(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
