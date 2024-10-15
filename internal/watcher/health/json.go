package health

import (
	"encoding/json"
	"time"

	"github.com/yusing/go-proxy/internal/net/types"
)

type JSONRepresentation struct {
	Name    string
	Config  *HealthCheckConfig
	Status  Status
	Started time.Time
	Uptime  time.Duration
	URL     types.URL
	Extra   map[string]any
}

func (jsonRepr *JSONRepresentation) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":    jsonRepr.Name,
		"config":  jsonRepr.Config,
		"started": jsonRepr.Started.Unix(),
		"status":  jsonRepr.Status.String(),
		"uptime":  jsonRepr.Uptime.Seconds(),
		"url":     jsonRepr.URL.String(),
		"extra":   jsonRepr.Extra,
	})
}
