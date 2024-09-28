package v1

import (
	"fmt"
	"net/http"
	"strings"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/config"
	R "github.com/yusing/go-proxy/internal/route"
)

func CheckHealth(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("target")
	if target == "" {
		U.HandleErr(w, r, U.ErrMissingKey("target"), http.StatusBadRequest)
		return
	}

	var ok bool
	route := cfg.FindRoute(target)

	switch {
	case route == nil:
		U.HandleErr(w, r, U.ErrNotFound("target", target), http.StatusNotFound)
		return
	case route.Type() == R.RouteTypeReverseProxy:
		ok = U.IsSiteHealthy(route.URL().String())
	case route.Type() == R.RouteTypeStream:
		entry := route.Entry()
		ok = U.IsStreamHealthy(
			strings.Split(entry.Scheme, ":")[1], // target scheme
			fmt.Sprintf("%s:%v", entry.Host, strings.Split(entry.Port, ":")[1]),
		)
	}

	if ok {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusRequestTimeout)
	}
}
