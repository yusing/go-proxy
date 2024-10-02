package v1

import (
	"net/http"
	"strings"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/utils"
)

const (
	ListRoutes          = "routes"
	ListConfigFiles     = "config_files"
	ListMiddlewares     = "middlewares"
	ListMiddlewareTrace = "middleware_trace"
	ListMatchDomains    = "match_domains"
	ListHomepageConfig  = "homepage_config"
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
	case ListMiddlewares:
		listMiddlewares(w, r)
	case ListMiddlewareTrace:
		listMiddlewareTrace(w, r)
	case ListMatchDomains:
		listMatchDomains(cfg, w, r)
	case ListHomepageConfig:
		listHomepageConfig(cfg, w, r)
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

	U.HandleErr(w, r, U.RespondJson(w, routes))
}

func listConfigFiles(w http.ResponseWriter, r *http.Request) {
	files, err := utils.ListFiles(common.ConfigBasePath, 1)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	for i := range files {
		files[i] = strings.TrimPrefix(files[i], common.ConfigBasePath+"/")
	}
	U.HandleErr(w, r, U.RespondJson(w, files))
}

func listMiddlewareTrace(w http.ResponseWriter, r *http.Request) {
	U.HandleErr(w, r, U.RespondJson(w, middleware.GetAllTrace()))
}

func listMiddlewares(w http.ResponseWriter, r *http.Request) {
	U.HandleErr(w, r, U.RespondJson(w, middleware.All()))
}

func listMatchDomains(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	U.HandleErr(w, r, U.RespondJson(w, cfg.Value().MatchDomains))
}

func listHomepageConfig(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	U.HandleErr(w, r, U.RespondJson(w, cfg.HomepageConfig()))
}
