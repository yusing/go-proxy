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
	ListFiles            = "files"
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
			http.NotFound(w, r)
			return
		} else {
			U.RespondJSON(w, r, route)
		}
	case ListRoutes:
		U.RespondJSON(w, r, config.RoutesByAlias(route.RouteType(r.FormValue("type"))))
	case ListFiles:
		listFiles(w, r)
	case ListMiddlewares:
		U.RespondJSON(w, r, middleware.All())
	case ListMiddlewareTraces:
		U.RespondJSON(w, r, middleware.GetAllTrace())
	case ListMatchDomains:
		U.RespondJSON(w, r, config.Value().MatchDomains)
	case ListHomepageConfig:
		U.RespondJSON(w, r, config.HomepageConfig())
	case ListTasks:
		U.RespondJSON(w, r, task.DebugTaskList())
	default:
		U.HandleErr(w, r, U.ErrInvalidKey("what"), http.StatusBadRequest)
	}
}

func listRoute(which string) any {
	if which == "" || which == "all" {
		return config.RoutesByAlias()
	}
	routes := config.RoutesByAlias()
	route, ok := routes[which]
	if !ok {
		return nil
	}
	return route
}

func listFiles(w http.ResponseWriter, r *http.Request) {
	files, err := utils.ListFiles(common.ConfigBasePath, 0)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	resp := map[FileType][]string{
		FileTypeConfig:     make([]string, 0),
		FileTypeProvider:   make([]string, 0),
		FileTypeMiddleware: make([]string, 0),
	}

	for _, file := range files {
		t := fileType(file)
		file = strings.TrimPrefix(file, common.ConfigBasePath+"/")
		resp[t] = append(resp[t], file)
	}

	mids, err := utils.ListFiles(common.MiddlewareComposeBasePath, 0)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	for _, mid := range mids {
		mid = strings.TrimPrefix(mid, common.MiddlewareComposeBasePath+"/")
		resp[FileTypeMiddleware] = append(resp[FileTypeMiddleware], mid)
	}
	U.RespondJSON(w, r, resp)
}
