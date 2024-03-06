package main

import "fmt"

type ProxyConfig struct {
	id          string
	Alias       string
	Scheme      string
	Host        string
	Port        string
	LoadBalance string
	Path        string // http proxy only
	PathMode    string // http proxy only
}

func NewProxyConfig() ProxyConfig {
	return ProxyConfig{}
}

func (cfg *ProxyConfig) UpdateId() {
	cfg.id = fmt.Sprintf("%s-%s-%s-%s-%s", cfg.Alias, cfg.Scheme, cfg.Host, cfg.Port, cfg.Path)
}
