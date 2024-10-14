package health

import (
	"time"

	"github.com/yusing/go-proxy/internal/common"
)

type HealthCheckConfig struct {
	Disabled bool          `json:"disabled" yaml:"disabled"`
	Path     string        `json:"path,omitempty" yaml:"path"`
	UseGet   bool          `json:"use_get,omitempty" yaml:"use_get"`
	Interval time.Duration `json:"interval" yaml:"interval"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout"`
}

func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Interval: common.HealthCheckIntervalDefault,
		Timeout:  common.HealthCheckTimeoutDefault,
	}
}
