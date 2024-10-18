package v1

import (
	"net/http"
	"strings"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/utils"
)

const (
	ListRoute            = "route"
	ListRoutes           = "routes"
	ListConfigFiles      = "config_files"
	ListMiddlewares      = "middlewares"
	ListMiddlewareTraces = "middleware_trace"
	ListMatchDomains     = "match_domains"
	ListHomepageConfig   = "homepage_config"
	ListTasks            = "tasks"
)

func List(w http.ResponseWriter, r *http.Request) {
	what := r.PathValue("what")
	if what == "" {
		what = ListRoutes
	}
	which := r.PathValue("which")

	switch what {
	case ListRoute:
		if route := listRoute(which); route == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		} else {
			U.RespondJSON(w, r, route)
		}
	case ListRoutes:
		U.RespondJSON(w, r, config.RoutesByAlias(route.RouteType(r.FormValue("type"))))
	case ListConfigFiles:
		listConfigFiles(w, r)
	case ListMiddlewares:
		U.RespondJSON(w, r, middleware.All())
	case ListMiddlewareTraces:
		U.RespondJSON(w, r, middleware.GetAllTrace())
	case ListMatchDomains:
		U.RespondJSON(w, r, config.Value().MatchDomains)
	case ListHomepageConfig:
		U.RespondJSON(w, r, config.HomepageConfig())
	case ListTasks:
		U.RespondJSON(w, r, task.DebugTaskMap())
	default:
		U.HandleErr(w, r, U.ErrInvalidKey("what"), http.StatusBadRequest)
	}
}

func listRoute(which string) any {
	if which == "" {
		which = "all"
	}
	if which == "all" {
		return config.RoutesByAlias()
	}
	routes := config.RoutesByAlias()
	route, ok := routes[which]
	if !ok {
		return nil
	}
	return route
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
	U.RespondJSON(w, r, files)
}
