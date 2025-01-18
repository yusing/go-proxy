package types

import (
	"context"

	"github.com/yusing/go-proxy/internal/net/http/accesslog"
	"github.com/yusing/go-proxy/internal/utils"

	E "github.com/yusing/go-proxy/internal/error"
)

type (
	Config struct {
		AutoCert        *AutoCertConfig `json:"autocert" validate:"omitempty"`
		Entrypoint      Entrypoint      `json:"entrypoint"`
		Providers       Providers       `json:"providers"`
		MatchDomains    []string        `json:"match_domains" validate:"dive,fqdn"`
		Homepage        HomepageConfig  `json:"homepage"`
		TimeoutShutdown int             `json:"timeout_shutdown" validate:"gte=0"`
	}
	Providers struct {
		Files        []string             `json:"include" validate:"dive,filepath"`
		Docker       map[string]string    `json:"docker" validate:"dive,unix_addr|url"`
		Notification []NotificationConfig `json:"notification"`
	}
	Entrypoint struct {
		Middlewares []map[string]any  `json:"middlewares"`
		AccessLog   *accesslog.Config `json:"access_log" validate:"omitempty"`
	}
	NotificationConfig map[string]any

	ConfigInstance interface {
		Value() *Config
		Reload() E.Error
		Statistics() map[string]any
		Context() context.Context
	}
)

func DefaultConfig() *Config {
	return &Config{
		TimeoutShutdown: 3,
		Homepage: HomepageConfig{
			UseDefaultCategories: true,
		},
	}
}

func Validate(data []byte) E.Error {
	var model Config
	return utils.DeserializeYAML(data, &model)
}

func init() {
	utils.RegisterDefaultValueFactory(DefaultConfig)
}
