package config

import (
	route "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/route/provider"
	"github.com/yusing/go-proxy/internal/route/types"
)

func (cfg *Config) DumpEntries() map[string]*types.RawEntry {
	entries := make(map[string]*types.RawEntry)
	cfg.providers.RangeAll(func(_ string, p *provider.Provider) {
		p.RangeRoutes(func(alias string, r *route.Route) {
			entries[alias] = r.Entry
		})
	})
	return entries
}

func (cfg *Config) DumpProviders() map[string]*provider.Provider {
	entries := make(map[string]*provider.Provider)
	cfg.providers.RangeAll(func(name string, p *provider.Provider) {
		entries[name] = p
	})
	return entries
}

func (cfg *Config) Statistics() map[string]any {
	nTotalStreams := 0
	nTotalRPs := 0
	providerStats := make(map[string]provider.ProviderStats)

	cfg.providers.RangeAll(func(name string, p *provider.Provider) {
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
