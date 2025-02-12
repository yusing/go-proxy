package period

import (
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/yusing/go-proxy/internal/api/v1/utils"
	metricsutils "github.com/yusing/go-proxy/internal/metrics/utils"
)

func (p *Poller[T, AggregateT]) lastResultHandler(w http.ResponseWriter, r *http.Request) {
	info := p.GetLastResult()
	if info == nil {
		http.Error(w, "no system info", http.StatusNoContent)
		return
	}
	utils.RespondJSON(w, r, info)
}

func (p *Poller[T, AggregateT]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	period := query.Get("period")
	if period == "" {
		p.lastResultHandler(w, r)
		return
	}
	periodFilter := Filter(period)
	if !periodFilter.IsValid() {
		http.Error(w, "invalid period", http.StatusBadRequest)
		return
	}
	rangeData := p.Get(periodFilter)
	if len(rangeData) == 0 {
		http.Error(w, "no data", http.StatusNoContent)
		return
	}
	if p.aggregator != nil {
		total, aggregated := p.aggregator(rangeData, query)
		utils.RespondJSON(w, r, map[string]any{
			"total": total,
			"data":  aggregated,
		})
	} else {
		utils.RespondJSON(w, r, rangeData)
	}
}

func (p *Poller[T, AggregateT]) ServeWS(allowedDomains []string, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	period := query.Get("period")
	interval := metricsutils.QueryDuration(query, "interval", 0)

	minInterval := 1 * time.Second
	if interval == 0 {
		interval = p.interval()
	}
	if interval < minInterval {
		interval = minInterval
	}

	if period == "" {
		utils.PeriodicWS(allowedDomains, w, r, interval, func(conn *websocket.Conn) error {
			return wsjson.Write(r.Context(), conn, p.GetLastResult())
		})
	} else {
		periodFilter := Filter(period)
		if !periodFilter.IsValid() {
			http.Error(w, "invalid period", http.StatusBadRequest)
			return
		}
		if p.aggregator != nil {
			utils.PeriodicWS(allowedDomains, w, r, interval, func(conn *websocket.Conn) error {
				total, aggregated := p.aggregator(p.Get(periodFilter), query)
				return wsjson.Write(r.Context(), conn, map[string]any{
					"total": total,
					"data":  aggregated,
				})
			})
		} else {
			utils.PeriodicWS(allowedDomains, w, r, interval, func(conn *websocket.Conn) error {
				return wsjson.Write(r.Context(), conn, p.Get(periodFilter))
			})
		}
	}
}
