package config

import (
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
