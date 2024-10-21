package types

type (
	Config struct {
		Providers       Providers      `json:"providers" yaml:",flow"`
		AutoCert        AutoCertConfig `json:"autocert" yaml:",flow"`
		ExplicitOnly    bool           `json:"explicit_only" yaml:"explicit_only"`
		MatchDomains    []string       `json:"match_domains" yaml:"match_domains"`
		TimeoutShutdown int            `json:"timeout_shutdown" yaml:"timeout_shutdown"`
		RedirectToHTTPS bool           `json:"redirect_to_https" yaml:"redirect_to_https"`
	}
	Providers struct {
		Files        []string              `json:"include" yaml:"include"`
		Docker       map[string]string     `json:"docker" yaml:"docker"`
		Notification NotificationConfigMap `json:"notification" yaml:"notification"`
	}
)

func DefaultConfig() *Config {
	return &Config{
		Providers:       Providers{},
		TimeoutShutdown: 3,
		RedirectToHTTPS: false,
	}
}
