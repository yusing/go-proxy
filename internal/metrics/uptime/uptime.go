package uptime

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/yusing/go-proxy/internal/metrics/period"
	metricsutils "github.com/yusing/go-proxy/internal/metrics/utils"
	"github.com/yusing/go-proxy/internal/route/routes/routequery"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type (
	StatusByAlias struct {
		Map       map[string]*routequery.HealthInfoRaw
		Timestamp time.Time
	}
	Status struct {
		Status    health.Status
		Latency   time.Duration
		Timestamp time.Time
	}
	RouteStatuses map[string][]*Status
	Aggregated    []map[string]any
)

var Poller = period.NewPollerWithAggregator("uptime", getStatuses, aggregateStatuses)

func init() {
	Poller.Start()
}

func getStatuses(ctx context.Context, _ *StatusByAlias) (*StatusByAlias, error) {
	return &StatusByAlias{
		Map:       routequery.HealthInfo(),
		Timestamp: time.Now(),
	}, nil
}

func aggregateStatuses(entries []*StatusByAlias, query url.Values) (int, Aggregated) {
	limit := metricsutils.QueryInt(query, "limit", 0)
	offset := metricsutils.QueryInt(query, "offset", 0)
	keyword := query.Get("keyword")

	statuses := make(RouteStatuses)
	for _, entry := range entries {
		for alias, status := range entry.Map {
			statuses[alias] = append(statuses[alias], &Status{
				Status:    status.Status,
				Latency:   status.Latency,
				Timestamp: entry.Timestamp,
			})
		}
	}
	if keyword != "" {
		for alias := range statuses {
			if !fuzzy.MatchFold(keyword, alias) {
				delete(statuses, alias)
			}
		}
	}
	return len(statuses), statuses.aggregate(limit, offset)
}

func (rs RouteStatuses) calculateInfo(statuses []*Status) (up float64, down float64, idle float64, latency int64) {
	if len(statuses) == 0 {
		return 0, 0, 0, 0
	}
	total := float64(0)
	for _, status := range statuses {
		// ignoring unknown; treating napping and starting as downtime
		if status.Status == health.StatusUnknown {
			continue
		}
		switch {
		case status.Status == health.StatusHealthy:
			up++
		case status.Status.Idling():
			idle++
		default:
			down++
		}
		total++
		latency += status.Latency.Milliseconds()
	}
	if total == 0 {
		return 0, 0, 0, 0
	}
	return up / total, down / total, idle / total, latency / int64(total)
}

func (rs RouteStatuses) aggregate(limit int, offset int) Aggregated {
	n := len(rs)
	beg, end, ok := metricsutils.CalculateBeginEnd(n, limit, offset)
	if !ok {
		return Aggregated{}
	}
	i := 0
	sortedAliases := make([]string, n)
	for alias := range rs {
		sortedAliases[i] = alias
		i++
	}
	sort.Strings(sortedAliases)
	sortedAliases = sortedAliases[beg:end]
	result := make(Aggregated, len(sortedAliases))
	for i, alias := range sortedAliases {
		statuses := rs[alias]
		up, down, idle, latency := rs.calculateInfo(statuses)
		result[i] = map[string]any{
			"alias":       alias,
			"uptime":      up,
			"downtime":    down,
			"idle":        idle,
			"avg_latency": latency,
			"statuses":    statuses,
		}
	}
	return result
}

func (s *Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"status":    s.Status.String(),
		"latency":   s.Latency.Milliseconds(),
		"timestamp": s.Timestamp.Unix(),
	})
}

func (s *StatusByAlias) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"statuses":  s.Map,
		"timestamp": s.Timestamp.Unix(),
	})
}
