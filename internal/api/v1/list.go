package v1

import (
	"encoding/json"
	"net/http"
	"os"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
)

const (
	ListRoutes          = "routes"
	ListConfigFiles     = "config_files"
	ListMiddlewareTrace = "middleware_trace"
)

func List(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	what := r.PathValue("what")
	if what == "" {
		what = ListRoutes
	}

	switch what {
	case ListRoutes:
		listRoutes(cfg, w, r)
	case ListConfigFiles:
		listConfigFiles(w, r)
	case ListMiddlewareTrace:
		listMiddlewareTrace(w, r)
	default:
		U.HandleErr(w, r, U.ErrInvalidKey("what"), http.StatusBadRequest)
	}
}

func listRoutes(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	routes := cfg.RoutesByAlias()
	typeFilter := r.FormValue("type")
	if typeFilter != "" {
		for k, v := range routes {
			if v["type"] != typeFilter {
				delete(routes, k)
			}
		}
	}

	if err := U.RespondJson(w, routes); err != nil {
		U.HandleErr(w, r, err)
	}
}

func listConfigFiles(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(common.ConfigBasePath)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	filenames := make([]string, len(files))
	for i, f := range files {
		filenames[i] = f.Name()
	}
	resp, err := json.Marshal(filenames)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	w.Write(resp)
}

func listMiddlewareTrace(w http.ResponseWriter, r *http.Request) {
	resp, err := json.Marshal(middleware.GetAllTrace())
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	w.Write(resp)
}
