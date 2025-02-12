package uptime

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yusing/go-proxy/internal/metrics/period"
	"github.com/yusing/go-proxy/internal/route/routes/routequery"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	Statuses struct {
		Statuses  map[string]health.Status
		Timestamp time.Time
	}
	Status struct {
		Status    health.Status
		Timestamp time.Time
	}
	Aggregated map[string][]Status
)

var Poller = period.NewPollerWithAggregator("uptime", 1*time.Second, getStatuses, aggregateStatuses)

func init() {
	Poller.Start()
}

func getStatuses(ctx context.Context) (*Statuses, error) {
	return &Statuses{
		Statuses:  routequery.HealthStatuses(),
		Timestamp: time.Now(),
	}, nil
}

func aggregateStatuses(entries ...*Statuses) any {
	aggregated := make(Aggregated)
	for _, entry := range entries {
		for alias, status := range entry.Statuses {
			aggregated[alias] = append(aggregated[alias], Status{
				Status:    status,
				Timestamp: entry.Timestamp,
			})
		}
	}
	return aggregated.finalize()
}

func (a Aggregated) calculateUptime(alias string) float64 {
	aggregated := a[alias]
	if len(aggregated) == 0 {
		return 0
	}
	uptime := 0
	for _, status := range aggregated {
		if status.Status == health.StatusHealthy {
			uptime++
		}
	}
	return float64(uptime) / float64(len(aggregated))
}

func (a Aggregated) finalize() map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{}, len(a))
	for alias, statuses := range a {
		result[alias] = map[string]interface{}{
			"uptime":   a.calculateUptime(alias),
			"statuses": statuses,
		}
	}
	return result
}

func (s *Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"status":    s.Status.String(),
		"timestamp": s.Timestamp.Unix(),
		"tooltip":   strutils.FormatTime(s.Timestamp),
	})
}

func (s *Statuses) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"statuses":  s.Statuses,
		"timestamp": s.Timestamp.Unix(),
		"tooltip":   strutils.FormatTime(s.Timestamp),
	})
}
