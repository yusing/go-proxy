package monitor

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type JSONRepresentation struct {
	Name     string
	Config   *health.HealthCheckConfig
	Status   health.Status
	Started  time.Time
	Uptime   time.Duration
	Latency  time.Duration
	LastSeen time.Time
	Detail   string
	URL      types.URL
	Extra    map[string]any
}

func (jsonRepr *JSONRepresentation) MarshalJSON() ([]byte, error) {
	url := jsonRepr.URL.String()
	if url == "http://:0" {
		url = ""
	}
	return json.Marshal(map[string]any{
		"name":        jsonRepr.Name,
		"config":      jsonRepr.Config,
		"started":     jsonRepr.Started.Unix(),
		"startedStr":  strutils.FormatTime(jsonRepr.Started),
		"status":      jsonRepr.Status.String(),
		"uptime":      jsonRepr.Uptime.Seconds(),
		"uptimeStr":   strutils.FormatDuration(jsonRepr.Uptime),
		"latency":     jsonRepr.Latency.Seconds(),
		"latencyStr":  strconv.Itoa(int(jsonRepr.Latency.Milliseconds())) + " ms",
		"lastSeen":    jsonRepr.LastSeen.Unix(),
		"lastSeenStr": strutils.FormatLastSeen(jsonRepr.LastSeen),
		"detail":      jsonRepr.Detail,
		"url":         url,
		"extra":       jsonRepr.Extra,
	})
}
