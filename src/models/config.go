package model

type Config struct {
	Providers       ProxyProviders `yaml:",flow" json:"providers"`
	AutoCert        AutoCertConfig `yaml:",flow" json:"autocert"`
	TimeoutShutdown int            `yaml:"timeout_shutdown" json:"timeout_shutdown"`
	RedirectToHTTPS bool           `yaml:"redirect_to_https" json:"redirect_to_https"`
}

func DefaultConfig() *Config {
	return &Config{
		Providers:       ProxyProviders{},
		TimeoutShutdown: 3,
		RedirectToHTTPS: true,
	}
}
