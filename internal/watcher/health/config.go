package health

import (
	"time"

	"github.com/yusing/go-proxy/internal/common"
)

type HealthCheckConfig struct {
	Disable  bool          `json:"disable,omitempty" aliases:"disabled"`
	Path     string        `json:"path,omitempty" validate:"omitempty,uri,startswith=/"`
	UseGet   bool          `json:"use_get,omitempty"`
	Interval time.Duration `json:"interval" validate:"omitempty,min=1s"`
	Timeout  time.Duration `json:"timeout" validate:"omitempty,min=1s"`
}

var DefaultHealthConfig = &HealthCheckConfig{
	Interval: common.HealthCheckIntervalDefault,
	Timeout:  common.HealthCheckTimeoutDefault,
}
