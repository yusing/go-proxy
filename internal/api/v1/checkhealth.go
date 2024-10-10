package v1

import (
	"fmt"
	"net/http"
	"strings"

	. "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/config"
	R "github.com/yusing/go-proxy/internal/route"
)

func CheckHealth(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("target")
	if target == "" {
		HandleErr(w, r, ErrMissingKey("target"), http.StatusBadRequest)
		return
	}

	var ok bool
	route := cfg.FindRoute(target)

	switch {
	case route == nil:
		HandleErr(w, r, ErrNotFound("target", target), http.StatusNotFound)
		return
	case route.Type() == R.RouteTypeReverseProxy:
		ok = IsSiteHealthy(route.URL().String())
	case route.Type() == R.RouteTypeStream:
		entry := route.Entry()
		ok = IsStreamHealthy(
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
