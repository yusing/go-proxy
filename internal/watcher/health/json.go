package health

import (
	"encoding/json"
	"time"

	"github.com/yusing/go-proxy/internal/net/types"
	U "github.com/yusing/go-proxy/internal/utils"
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
		"name":       jsonRepr.Name,
		"config":     jsonRepr.Config,
		"started":    jsonRepr.Started.Unix(),
		"startedStr": U.FormatTime(jsonRepr.Started),
		"status":     jsonRepr.Status.String(),
		"uptime":     jsonRepr.Uptime.Seconds(),
		"uptimeStr":  U.FormatDuration(jsonRepr.Uptime),
		"url":        jsonRepr.URL,
		"extra":      jsonRepr.Extra,
	})
}
