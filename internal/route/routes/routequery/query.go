package routequery

import (
	"strings"
	"time"

	"github.com/yusing/go-proxy/internal"
	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/route/entry"
	provider "github.com/yusing/go-proxy/internal/route/provider/types"
	"github.com/yusing/go-proxy/internal/route/routes"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func getHealthInfo(r route.Route) map[string]string {
	mon := r.HealthMonitor()
	if mon == nil {
		return map[string]string{
			"status":  "unknown",
			"uptime":  "n/a",
			"latency": "n/a",
		}
	}
	return map[string]string{
		"status":  mon.Status().String(),
		"uptime":  mon.Uptime().Round(time.Second).String(),
		"latency": mon.Latency().Round(time.Microsecond).String(),
	}
}

func HealthMap() map[string]map[string]string {
	healthMap := make(map[string]map[string]string)
	routes.GetHTTPRoutes().RangeAll(func(alias string, r route.HTTPRoute) {
		healthMap[alias] = getHealthInfo(r)
	})
	routes.GetStreamRoutes().RangeAll(func(alias string, r route.StreamRoute) {
		healthMap[alias] = getHealthInfo(r)
	})
	return healthMap
}

func HomepageCategories() []string {
	check := make(map[string]struct{})
	categories := make([]string, 0)
	routes.GetHTTPRoutes().RangeAll(func(alias string, r route.HTTPRoute) {
		en := r.RawEntry()
		if en.Homepage.IsEmpty() || en.Homepage.Category == "" {
			return
		}
		if _, ok := check[en.Homepage.Category]; ok {
			return
		}
		check[en.Homepage.Category] = struct{}{}
		categories = append(categories, en.Homepage.Category)
	})
	return categories
}

func HomepageConfig(useDefaultCategories bool, categoryFilter, providerFilter string) homepage.Config {
	hpCfg := homepage.NewHomePageConfig()

	routes.GetHTTPRoutes().RangeAll(func(alias string, r route.HTTPRoute) {
		en := r.RawEntry()
		item := en.Homepage

		if item.IsEmpty() {
			item = homepage.NewItem(alias)
		}

		if override := item.GetOverride(); override != item {
			if providerFilter != "" && override.Provider != providerFilter ||
				categoryFilter != "" && override.Category != categoryFilter {
				return
			}
			hpCfg.Add(override)
			return
		}

		item.Alias = alias
		item.Provider = r.RawEntry().Provider

		if providerFilter != "" && item.Provider != providerFilter {
			return
		}

		if item.Name == "" {
			reference := r.TargetName()
			cont := r.RawEntry().Container
			if cont != nil {
				reference = cont.ImageName
			}
			name, ok := internal.GetDisplayName(reference)
			if ok {
				item.Name = name
			} else {
				item.Name = strutils.Title(
					strings.ReplaceAll(
						strings.ReplaceAll(alias, "-", " "),
						"_", " ",
					),
				)
			}
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

		if categoryFilter != "" && item.Category != categoryFilter {
			return
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

func RoutesByAlias(typeFilter ...route.RouteType) map[string]route.Route {
	rts := make(map[string]route.Route)
	if len(typeFilter) == 0 || typeFilter[0] == "" {
		typeFilter = []route.RouteType{route.RouteTypeReverseProxy, route.RouteTypeStream}
	}
	for _, t := range typeFilter {
		switch t {
		case route.RouteTypeReverseProxy:
			routes.GetHTTPRoutes().RangeAll(func(alias string, r route.HTTPRoute) {
				rts[alias] = r
			})
		case route.RouteTypeStream:
			routes.GetStreamRoutes().RangeAll(func(alias string, r route.StreamRoute) {
				rts[alias] = r
			})
		}
	}
	return rts
}
