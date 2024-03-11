package main

import "fmt"

type ProxyConfig struct {
	Alias       string
	Scheme      string
	Host        string
	Port        string
	LoadBalance string // docker provider only
	Path        string // http proxy only
	PathMode    string `yaml:"path_mode"` // http proxy only

	provider *Provider
}

func NewProxyConfig(provider *Provider) ProxyConfig {
	return ProxyConfig{
		provider: provider,
	}
}

// used by `GetFileProxyConfigs`
func (cfg *ProxyConfig) SetDefaults() error {
	if cfg.Alias == "" {
		return fmt.Errorf("alias is required")
	}
	if cfg.Scheme == "" {
		cfg.Scheme = "http"
	}
	if cfg.Host == "" {
		return fmt.Errorf("host is required for %q", cfg.Alias)
	}
	if cfg.Port == "" {
		cfg.Port = "80"
	}
	return nil
}

func (cfg *ProxyConfig) GetID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", cfg.Alias, cfg.Scheme, cfg.Host, cfg.Port, cfg.Path)
}
