package config

import (
	"fmt"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	H "github.com/yusing/go-proxy/internal/homepage"
	M "github.com/yusing/go-proxy/internal/models"
	PR "github.com/yusing/go-proxy/internal/proxy/provider"
	R "github.com/yusing/go-proxy/internal/route"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

func (cfg *Config) DumpEntries() map[string]*M.RawEntry {
	entries := make(map[string]*M.RawEntry)
	cfg.forEachRoute(func(alias string, r R.Route, p *PR.Provider) {
		entries[alias] = r.Entry()
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

func (cfg *Config) HomepageConfig() H.HomePageConfig {
	var proto, port string
	domains := cfg.value.MatchDomains
	cert, _ := cfg.autocertProvider.GetCert(nil)
	if cert != nil {
		proto = "https"
		port = common.ProxyHTTPPort
	} else {
		proto = "http"
		port = common.ProxyHTTPSPort
	}

	hpCfg := H.NewHomePageConfig()
	cfg.forEachRoute(func(alias string, r R.Route, p *PR.Provider) {
		if !r.Started() {
			return
		}

		entry := r.Entry()
		item := entry.Homepage
		if item == nil {
			item = &H.HomePageItem{}
		}

		if item.Name == "" {
			item.Name = U.Title(alias)
		}

		if p.GetType() == PR.ProviderTypeDocker {
			if item.Category == "" {
				item.Category = "Docker"
			}
			item.SourceType = string(PR.ProviderTypeDocker)
		} else if p.GetType() == PR.ProviderTypeFile {
			if item.Category == "" {
				item.Category = "Others"
			}
			item.SourceType = string(PR.ProviderTypeFile)
		}

		if item.URL == "" && r.Type() == R.RouteTypeReverseProxy {
			if len(domains) > 0 {
				item.URL = fmt.Sprintf("%s://%s.%s:%s", proto, strings.ToLower(alias), domains[0], port)
			}
		}
		hpCfg.Add(item)
	})
	return hpCfg
}

func (cfg *Config) RoutesByAlias() map[string]U.SerializedObject {
	routes := make(map[string]U.SerializedObject)
	cfg.forEachRoute(func(alias string, r R.Route, p *PR.Provider) {
		if !r.Started() {
			return
		}
		obj, err := U.Serialize(r)
		if err.HasError() {
			cfg.l.Error(err)
			return
		}
		obj["provider"] = p.GetName()
		obj["type"] = string(r.Type())
		obj["started"] = r.Started()
		obj["raw"] = r.Entry()
		routes[alias] = obj
	})
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

func (cfg *Config) FindRoute(alias string) R.Route {
	return F.MapFind(cfg.proxyProviders,
		func(p *PR.Provider) (R.Route, bool) {
			if route, ok := p.GetRoute(alias); ok {
				return route, true
			}
			return nil, false
		},
	)
}
