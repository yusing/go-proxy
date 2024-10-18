package config

import (
	"fmt"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/homepage"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	"github.com/yusing/go-proxy/internal/route"
	proxy "github.com/yusing/go-proxy/internal/route/provider"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

func DumpEntries() map[string]*entry.RawEntry {
	entries := make(map[string]*entry.RawEntry)
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
	route.GetReverseProxies().RangeAll(func(alias string, r *route.HTTPRoute) {
		en := r.Raw
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
			item.Name = U.Title(
				strings.ReplaceAll(
					strings.ReplaceAll(alias, "-", " "),
					"_", " ",
				),
			)
		}

		if entry.IsDocker(r) {
			if item.Category == "" {
				item.Category = "Docker"
			}
			item.SourceType = string(proxy.ProviderTypeDocker)
		} else if entry.UseLoadBalance(r) {
			if item.Category == "" {
				item.Category = "Load-balanced"
			}
			item.SourceType = "loadbalancer"
		} else {
			if item.Category == "" {
				item.Category = "Others"
			}
			item.SourceType = string(proxy.ProviderTypeFile)
		}

		if item.URL == "" {
			if len(domains) > 0 {
				item.URL = fmt.Sprintf("%s://%s.%s:%s", proto, strings.ToLower(alias), domains[0], port)
			}
		}
		item.AltURL = r.TargetURL().String()

		hpCfg.Add(item)
	})
	return hpCfg
}

func RoutesByAlias(typeFilter ...route.RouteType) map[string]any {
	routes := make(map[string]any)
	if len(typeFilter) == 0 || typeFilter[0] == "" {
		typeFilter = []route.RouteType{route.RouteTypeReverseProxy, route.RouteTypeStream}
	}
	for _, t := range typeFilter {
		switch t {
		case route.RouteTypeReverseProxy:
			route.GetReverseProxies().RangeAll(func(alias string, r *route.HTTPRoute) {
				routes[alias] = r
			})
		case route.RouteTypeStream:
			route.GetStreamProxies().RangeAll(func(alias string, r *route.StreamRoute) {
				routes[alias] = r
			})
		}
	}
	return routes
}

func Statistics() map[string]any {
	nTotalStreams := 0
	nTotalRPs := 0
	providerStats := make(map[string]proxy.ProviderStats)

	instance.providers.RangeAll(func(name string, p *proxy.Provider) {
		providerStats[name] = p.Statistics()
	})

	for _, stats := range providerStats {
		nTotalRPs += stats.NumRPs
		nTotalStreams += stats.NumStreams
	}

	return map[string]any{
		"num_total_streams":         nTotalStreams,
		"num_total_reverse_proxies": nTotalRPs,
		"providers":                 providerStats,
	}
}

func FindRoute(alias string) *route.Route {
	return F.MapFind(instance.providers,
		func(p *proxy.Provider) (*route.Route, bool) {
			if route, ok := p.GetRoute(alias); ok {
				return route, true
			}
			return nil, false
		},
	)
}
