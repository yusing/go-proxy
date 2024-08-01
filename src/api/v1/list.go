package v1

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/yusing/go-proxy/common"
	"github.com/yusing/go-proxy/config"

	U "github.com/yusing/go-proxy/api/v1/utils"
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
	type_filter := r.FormValue("type")
	if type_filter != "" {
		for k, v := range routes {
			if v["type"] != type_filter {
				delete(routes, k)
			}
		}
	}

	if err := U.RespondJson(routes, w); err != nil {
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
