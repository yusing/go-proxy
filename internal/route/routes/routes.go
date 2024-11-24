package routes

import (
	"github.com/yusing/go-proxy/internal/route/types"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

var (
	httpRoutes   = F.NewMapOf[string, types.HTTPRoute]()
	streamRoutes = F.NewMapOf[string, types.StreamRoute]()
)

func GetHTTPRoutes() F.Map[string, types.HTTPRoute] {
	return httpRoutes
}

func GetStreamRoutes() F.Map[string, types.StreamRoute] {
	return streamRoutes
}

func GetHTTPRoute(alias string) (types.HTTPRoute, bool) {
	return httpRoutes.Load(alias)
}

func GetStreamRoute(alias string) (types.StreamRoute, bool) {
	return streamRoutes.Load(alias)
}

func SetHTTPRoute(alias string, r types.HTTPRoute) {
	httpRoutes.Store(alias, r)
}

func SetStreamRoute(alias string, r types.StreamRoute) {
	streamRoutes.Store(alias, r)
}

func DeleteHTTPRoute(alias string) {
	httpRoutes.Delete(alias)
}

func DeleteStreamRoute(alias string) {
	streamRoutes.Delete(alias)
}
