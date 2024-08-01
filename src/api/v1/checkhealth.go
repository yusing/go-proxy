package v1

import (
	"fmt"
	"net/http"

	U "github.com/yusing/go-proxy/api/v1/utils"
	"github.com/yusing/go-proxy/config"
	R "github.com/yusing/go-proxy/route"
)

func CheckHealth(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("target")
	if target == "" {
		U.HandleErr(w, r, U.ErrMissingKey("target"), http.StatusBadRequest)
		return
	}

	var ok bool

	switch route := cfg.FindRoute(target).(type) {
	case nil:
		U.HandleErr(w, r, U.ErrNotFound("target", target), http.StatusNotFound)
		return
	case *R.HTTPRoute:
		path := r.FormValue("path")
		if path == "" {
			U.HandleErr(w, r, U.ErrMissingKey("path"), http.StatusBadRequest)
			return
		}
		sr, hasSr := route.GetSubroute(path)
		if !hasSr {
			U.HandleErr(w, r, U.ErrNotFound("path", path), http.StatusNotFound)
			return
		}
		ok = U.IsSiteHealthy(sr.TargetURL.String())
	case *R.StreamRoute:
		ok = U.IsStreamHealthy(
			route.Scheme.ProxyScheme.String(),
			fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort),
		)
	}

	if ok {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusRequestTimeout)
	}
}
