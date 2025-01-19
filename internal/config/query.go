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
	cfg.providers.RangeAll(func(_ string, p *provider.Provider) {
		entries[p.ShortName()] = p
	})
	return entries
}

func (cfg *Config) Statistics() map[string]any {
	var rps, streams provider.RouteStats
	var total uint16
	providerStats := make(map[string]provider.ProviderStats)

	cfg.providers.RangeAll(func(_ string, p *provider.Provider) {
		stats := p.Statistics()
		providerStats[p.ShortName()] = stats
		rps.AddOther(stats.RPs)
		streams.AddOther(stats.Streams)
		total += stats.RPs.Total + stats.Streams.Total
	})

	return map[string]any{
		"total":           total,
		"reverse_proxies": rps,
		"streams":         streams,
		"providers":       providerStats,
	}
}
