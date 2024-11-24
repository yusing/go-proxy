package config

import (
	"fmt"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/homepage"
	route "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/route/entry"
	proxy "github.com/yusing/go-proxy/internal/route/provider"
	"github.com/yusing/go-proxy/internal/route/routes"
	"github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func DumpEntries() map[string]*types.RawEntry {
	entries := make(map[string]*types.RawEntry)
	instance.providers.RangeAll(func(_ string, p *proxy.Provider) {
		p.RangeRoutes(func(alias string, r *route.Route) {
			entries[alias] = r.Entry
		})
	})
	return entries
}

func DumpProviders() map[string]*proxy.Provider {
	entries := make(map[string]*proxy.Provider)
	instance.providers.RangeAll(func(name string, p *proxy.Provider) {
		entries[name] = p
	})
	return entries
}

func HomepageConfig() homepage.Config {
	var proto, port string
	domains := instance.value.MatchDomains
	cert, _ := instance.autocertProvider.GetCert(nil)
	if cert != nil {
		proto = "https"
		port = common.ProxyHTTPSPort
	} else {
		proto = "http"
		port = common.ProxyHTTPPort
	}

	hpCfg := homepage.NewHomePageConfig()
	routes.GetHTTPRoutes().RangeAll(func(alias string, r types.HTTPRoute) {
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

		if item.Name == "" {
			item.Name = strutils.Title(
				strings.ReplaceAll(
					strings.ReplaceAll(alias, "-", " "),
					"_", " ",
				),
			)
		}

		if instance.value.Homepage.UseDefaultCategories {
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
			item.SourceType = string(proxy.ProviderTypeDocker)
		case entry.UseLoadBalance(r):
			if item.Category == "" {
				item.Category = "Load-balanced"
			}
			item.SourceType = "loadbalancer"
		default:
			if item.Category == "" {
				item.Category = "Others"
			}
			item.SourceType = string(proxy.ProviderTypeFile)
		}

		if item.URL == "" {
			if len(domains) > 0 {
				item.URL = fmt.Sprintf("%s://%s%s:%s", proto, strings.ToLower(alias), domains[0], port)
			}
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
			routes.GetHTTPRoutes().RangeAll(func(alias string, r types.HTTPRoute) {
				rts[alias] = r
			})
		case route.RouteTypeStream:
			routes.GetStreamRoutes().RangeAll(func(alias string, r types.StreamRoute) {
				rts[alias] = r
			})
		}
	}
	return rts
}

func Statistics() map[string]any {
	nTotalStreams := 0
	nTotalRPs := 0
	providerStats := make(map[string]proxy.ProviderStats)

	instance.providers.RangeAll(func(name string, p *proxy.Provider) {
		stats := p.Statistics()
		providerStats[name] = stats

		nTotalRPs += stats.NumRPs
		nTotalStreams += stats.NumStreams
	})

	return map[string]any{
		"num_total_streams":         nTotalStreams,
		"num_total_reverse_proxies": nTotalRPs,
		"providers":                 providerStats,
	}
}
