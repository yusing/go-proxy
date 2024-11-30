package types

type (
	Config struct {
		AutoCert        AutoCertConfig `json:"autocert" yaml:",flow"`
		Entrypoint      Entrypoint     `json:"entrypoint" yaml:",flow"`
		Providers       Providers      `json:"providers" yaml:",flow"`
		MatchDomains    []string       `json:"match_domains" yaml:"match_domains"`
		Homepage        HomepageConfig `json:"homepage" yaml:"homepage"`
		TimeoutShutdown int            `json:"timeout_shutdown" yaml:"timeout_shutdown"`
	}
	Providers struct {
		Files        []string             `json:"include" yaml:"include"`
		Docker       map[string]string    `json:"docker" yaml:"docker"`
		Notification []NotificationConfig `json:"notification" yaml:"notification"`
	}
	Entrypoint struct {
		RedirectToHTTPS bool `json:"redirect_to_https" yaml:"redirect_to_https"`
		Middlewares     []map[string]any
	}
	NotificationConfig map[string]any
)

func DefaultConfig() *Config {
	return &Config{
		TimeoutShutdown: 3,
		Homepage: HomepageConfig{
			UseDefaultCategories: true,
		},
		Entrypoint: Entrypoint{
			RedirectToHTTPS: false,
		},
	}
}
