package config

import (
	"fmt"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/homepage"
	PR "github.com/yusing/go-proxy/internal/proxy/provider"
	R "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/types"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

func (cfg *Config) DumpEntries() map[string]*types.RawEntry {
	entries := make(map[string]*types.RawEntry)
	cfg.forEachRoute(func(alias string, r *R.Route, p *PR.Provider) {
		entries[alias] = r.Entry
	})
	return entries
}

func (cfg *Config) DumpProviders() map[string]*PR.Provider {
	entries := make(map[string]*PR.Provider)
	cfg.proxyProviders.RangeAll(func(name string, p *PR.Provider) {
		entries[name] = p
	})
	return entries
}

func (cfg *Config) HomepageConfig() homepage.Config {
	var proto, port string
	domains := cfg.value.MatchDomains
	cert, _ := cfg.autocertProvider.GetCert(nil)
	if cert != nil {
		proto = "https"
		port = common.ProxyHTTPSPort
	} else {
		proto = "http"
		port = common.ProxyHTTPPort
	}

	hpCfg := homepage.NewHomePageConfig()
	R.GetReverseProxies().RangeAll(func(alias string, r *R.HTTPRoute) {
		entry := r.Raw
		item := entry.Homepage
		if item == nil {
			item = new(homepage.Item)
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

		if r.IsDocker() {
			if item.Category == "" {
				item.Category = "Docker"
			}
			item.SourceType = string(PR.ProviderTypeDocker)
		} else if r.UseLoadBalance() {
			if item.Category == "" {
				item.Category = "Load-balanced"
			}
			item.SourceType = "loadbalancer"
		} else {
			if item.Category == "" {
				item.Category = "Others"
			}
			item.SourceType = string(PR.ProviderTypeFile)
		}

		if item.URL == "" {
			if len(domains) > 0 {
				item.URL = fmt.Sprintf("%s://%s.%s:%s", proto, strings.ToLower(alias), domains[0], port)
			}
		}
		item.AltURL = r.URL().String()

		hpCfg.Add(item)
	})
	return hpCfg
}

func (cfg *Config) RoutesByAlias(typeFilter ...R.RouteType) map[string]any {
	routes := make(map[string]any)
	if len(typeFilter) == 0 || typeFilter[0] == "" {
		typeFilter = []R.RouteType{R.RouteTypeReverseProxy, R.RouteTypeStream}
	}
	for _, t := range typeFilter {
		switch t {
		case R.RouteTypeReverseProxy:
			R.GetReverseProxies().RangeAll(func(alias string, r *R.HTTPRoute) {
				routes[alias] = r
			})
		case R.RouteTypeStream:
			R.GetStreamProxies().RangeAll(func(alias string, r *R.StreamRoute) {
				routes[alias] = r
			})
		}
	}
	return routes
}

func (cfg *Config) Statistics() map[string]any {
	nTotalStreams := 0
	nTotalRPs := 0
	providerStats := make(map[string]PR.ProviderStats)

	cfg.proxyProviders.RangeAll(func(name string, p *PR.Provider) {
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

func (cfg *Config) FindRoute(alias string) *R.Route {
	return F.MapFind(cfg.proxyProviders,
		func(p *PR.Provider) (*R.Route, bool) {
			if route, ok := p.GetRoute(alias); ok {
				return route, true
			}
			return nil, false
		},
	)
}
