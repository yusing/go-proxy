package types

type Config struct {
	Providers       ProxyProviders `json:"providers" yaml:",flow"`
	AutoCert        AutoCertConfig `json:"autocert" yaml:",flow"`
	ExplicitOnly    bool           `json:"explicit_only" yaml:"explicit_only"`
	MatchDomains    []string       `json:"match_domains" yaml:"match_domains"`
	TimeoutShutdown int            `json:"timeout_shutdown" yaml:"timeout_shutdown"`
	RedirectToHTTPS bool           `json:"redirect_to_https" yaml:"redirect_to_https"`
}

func DefaultConfig() *Config {
	return &Config{
		Providers:       ProxyProviders{},
		TimeoutShutdown: 3,
		RedirectToHTTPS: false,
	}
}
