package health

import (
	"time"
)

type HealthCheckConfig struct {
	Disable  bool          `json:"disable,omitempty" yaml:"disable" aliases:"disabled"`
	Path     string        `json:"path,omitempty" yaml:"path"`
	UseGet   bool          `json:"use_get,omitempty" yaml:"use_get"`
	Interval time.Duration `json:"interval" yaml:"interval"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout"`
}
