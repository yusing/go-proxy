package uptime

import (
	"context"
	"time"

	"github.com/yusing/go-proxy/internal/metrics/period"
	"github.com/yusing/go-proxy/internal/route/routes/routequery"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	Statuses struct {
		Statuses  map[string]health.Status
		Timestamp int64
	}
	Status struct {
		Status    health.Status
		Timestamp int64
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
		Timestamp: time.Now().Unix(),
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
