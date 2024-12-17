package health

import (
	"time"
)

type HealthCheckConfig struct {
	Disable  bool          `json:"disable,omitempty" aliases:"disabled"`
	Path     string        `json:"path,omitempty" validate:"omitempty,uri,startswith=/"`
	UseGet   bool          `json:"use_get,omitempty"`
	Interval time.Duration `json:"interval" validate:"omitempty,min=1s"`
	Timeout  time.Duration `json:"timeout" validate:"omitempty,min=1s"`
}
