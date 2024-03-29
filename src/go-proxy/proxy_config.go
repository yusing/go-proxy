package main

import "fmt"

type ProxyConfig struct {
	Alias       string `yaml:"-" json:"-"`
	Scheme      string `yaml:"scheme" json:"scheme"`
	Host        string `yaml:"host" json:"host"`
	Port        string `yaml:"port" json:"port"`
	LoadBalance string `yaml:"-" json:"-"`                         // docker provider only
	NoTLSVerify bool   `yaml:"no_tls_verify" json:"no_tls_verify"` // http proxy only
	Path        string `yaml:"path" json:"path"`                   // http proxy only
	PathMode    string `yaml:"path_mode" json:"path_mode"`         // http proxy only

	provider *Provider
}

type ProxyConfigMap map[string]ProxyConfig
type ProxyConfigSlice []ProxyConfig

func NewProxyConfig(provider *Provider) ProxyConfig {
	return ProxyConfig{
		provider: provider,
	}
}

// used by `GetFileProxyConfigs`
func (cfg *ProxyConfig) SetDefaults() error {
	err := NewNestedError("invalid proxy config").Subject(cfg.Alias)

	if cfg.Alias == "" {
		err.Extra("alias is required")
	}
	if cfg.Scheme == "" {
		cfg.Scheme = "http"
	}
	if cfg.Host == "" {
		err.Extra("host is required")
	}
	if cfg.Port == "" {
		switch cfg.Scheme {
		case "http":
			cfg.Port = "80"
		case "https":
			cfg.Port = "443"
		default:
			err.Extraf("port is required for %s scheme", cfg.Scheme)
		}
	}
	if err.HasExtras() {
		return err
	}
	return nil
}

func (cfg *ProxyConfig) GetID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", cfg.Alias, cfg.Scheme, cfg.Host, cfg.Port, cfg.Path)
}
