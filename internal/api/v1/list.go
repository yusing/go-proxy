package v1

import (
	"encoding/json"
	"net/http"
	"os"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
)

func List(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	what := r.PathValue("what")
	if what == "" {
		what = "routes"
	}

	switch what {
	case "routes":
		listRoutes(cfg, w, r)
	case "config_files":
		listConfigFiles(w, r)
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
