package routes

import (
	"strings"

	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/route/entry"
	provider "github.com/yusing/go-proxy/internal/route/provider/types"
	"github.com/yusing/go-proxy/internal/route/types"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func HomepageConfig(useDefaultCategories bool) homepage.Config {
	hpCfg := homepage.NewHomePageConfig()
	GetHTTPRoutes().RangeAll(func(alias string, r types.HTTPRoute) {
		en := r.RawEntry()
		item := en.Homepage
		if item == nil {
			item = new(homepage.Item)
			item.Show = true
		}

		if !item.IsEmpty() {
			item.Show = true
		}

		if !item.Show {
			return
		}

		item.Alias = alias

		if item.Name == "" {
			item.Name = strutils.Title(
				strings.ReplaceAll(
					strings.ReplaceAll(alias, "-", " "),
					"_", " ",
				),
			)
		}

		if useDefaultCategories {
			if en.Container != nil && item.Category == "" {
				if category, ok := homepage.PredefinedCategories[en.Container.ImageName]; ok {
					item.Category = category
				}
			}

			if item.Category == "" {
				if category, ok := homepage.PredefinedCategories[strings.ToLower(alias)]; ok {
					item.Category = category
				}
			}
		}

		switch {
		case entry.IsDocker(r):
			if item.Category == "" {
				item.Category = "Docker"
			}
			item.SourceType = string(provider.ProviderTypeDocker)
		case entry.UseLoadBalance(r):
			if item.Category == "" {
				item.Category = "Load-balanced"
			}
			item.SourceType = "loadbalancer"
		default:
			if item.Category == "" {
				item.Category = "Others"
			}
			item.SourceType = string(provider.ProviderTypeFile)
		}

		item.AltURL = r.TargetURL().String()
		hpCfg.Add(item)
	})
	return hpCfg
}

func RoutesByAlias(typeFilter ...route.RouteType) map[string]any {
	rts := make(map[string]any)
	if len(typeFilter) == 0 || typeFilter[0] == "" {
		typeFilter = []route.RouteType{route.RouteTypeReverseProxy, route.RouteTypeStream}
	}
	for _, t := range typeFilter {
		switch t {
		case route.RouteTypeReverseProxy:
			GetHTTPRoutes().RangeAll(func(alias string, r types.HTTPRoute) {
				rts[alias] = r
			})
		case route.RouteTypeStream:
			GetStreamRoutes().RangeAll(func(alias string, r types.StreamRoute) {
				rts[alias] = r
			})
		}
	}
	return rts
}
