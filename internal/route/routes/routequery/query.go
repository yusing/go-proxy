package routequery

import (
	"strings"
	"time"

	"github.com/yusing/go-proxy/internal"
	"github.com/yusing/go-proxy/internal/homepage"
	provider "github.com/yusing/go-proxy/internal/route/provider/types"
	"github.com/yusing/go-proxy/internal/route/routes"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"github.com/yusing/go-proxy/internal/watcher/health"
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

func getHealthInfoRaw(r route.Route) map[string]any {
	mon := r.HealthMonitor()
	if mon == nil {
		return map[string]any{
			"status":  health.StatusUnknown,
			"latency": time.Duration(0),
		}
	}
	return map[string]any{
		"status":  mon.Status(),
		"latency": mon.Latency(),
	}
}

func HealthMap() map[string]map[string]string {
	healthMap := make(map[string]map[string]string, routes.NumRoutes())
	routes.RangeRoutes(func(alias string, r route.Route) {
		healthMap[alias] = getHealthInfo(r)
	})
	return healthMap
}

func HealthInfo() map[string]map[string]any {
	healthMap := make(map[string]map[string]any, routes.NumRoutes())
	routes.RangeRoutes(func(alias string, r route.Route) {
		healthMap[alias] = getHealthInfoRaw(r)
	})
	return healthMap
}

func HomepageCategories() []string {
	check := make(map[string]struct{})
	categories := make([]string, 0)
	routes.GetHTTPRoutes().RangeAll(func(alias string, r route.HTTPRoute) {
		homepage := r.HomepageConfig()
		if homepage.IsEmpty() || homepage.Category == "" {
			return
		}
		if _, ok := check[homepage.Category]; ok {
			return
		}
		check[homepage.Category] = struct{}{}
		categories = append(categories, homepage.Category)
	})
	return categories
}

func HomepageConfig(useDefaultCategories bool, categoryFilter, providerFilter string) homepage.Categories {
	hpCfg := homepage.NewHomePageConfig()

	routes.GetHTTPRoutes().RangeAll(func(alias string, r route.HTTPRoute) {
		item := r.HomepageConfig()

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
		item.Provider = r.ProviderName()

		if providerFilter != "" && item.Provider != providerFilter {
			return
		}

		if item.Name == "" {
			reference := r.TargetName()
			cont := r.ContainerInfo()
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
			container := r.ContainerInfo()
			if container != nil && item.Category == "" {
				if category, ok := homepage.PredefinedCategories[container.ImageName]; ok {
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
		case r.IsDocker():
			if item.Category == "" {
				item.Category = "Docker"
			}
			if r.IsAgent() {
				item.SourceType = string(provider.ProviderTypeAgent)
			} else {
				item.SourceType = string(provider.ProviderTypeDocker)
			}
		case r.UseLoadBalance():
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
		typeFilter = []route.RouteType{route.RouteTypeHTTP, route.RouteTypeStream}
	}
	for _, t := range typeFilter {
		switch t {
		case route.RouteTypeHTTP:
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
