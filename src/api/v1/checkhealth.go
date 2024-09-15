package v1

import (
	"fmt"
	"net/http"

	U "github.com/yusing/go-proxy/api/v1/utils"
	"github.com/yusing/go-proxy/config"
	PT "github.com/yusing/go-proxy/proxy/fields"
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
		path, err := PT.NewPath(r.FormValue("path"))
		if err.IsNotNil() {
			U.HandleErr(w, r, err, http.StatusBadRequest)
			return
		}
		sr, hasSr := route.GetSubroute(path)
		if !hasSr {
			U.HandleErr(w, r, U.ErrNotFound("path", string(path)), http.StatusNotFound)
			return
		}
		ok = U.IsSiteHealthy(sr.TargetURL.String())
	case *R.StreamRoute:
		ok = U.IsStreamHealthy(
			string(route.Scheme.ProxyScheme),
			fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort),
		)
	}

	if ok {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusRequestTimeout)
	}
}
