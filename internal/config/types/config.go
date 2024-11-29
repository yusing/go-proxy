package types

type (
	Config struct {
		Providers       Providers      `json:"providers" yaml:",flow"`
		AutoCert        AutoCertConfig `json:"autocert" yaml:",flow"`
		ExplicitOnly    bool           `json:"explicit_only" yaml:"explicit_only"`
		MatchDomains    []string       `json:"match_domains" yaml:"match_domains"`
		Homepage        HomepageConfig `json:"homepage" yaml:"homepage"`
		TimeoutShutdown int            `json:"timeout_shutdown" yaml:"timeout_shutdown"`
		RedirectToHTTPS bool           `json:"redirect_to_https" yaml:"redirect_to_https"`
	}
	Providers struct {
		Files        []string             `json:"include" yaml:"include"`
		Docker       map[string]string    `json:"docker" yaml:"docker"`
		Notification []NotificationConfig `json:"notification" yaml:"notification"`
	}
	NotificationConfig map[string]any
)

func DefaultConfig() *Config {
	return &Config{
		TimeoutShutdown: 3,
		Homepage: HomepageConfig{
			UseDefaultCategories: true,
		},
		RedirectToHTTPS: false,
	}
}
