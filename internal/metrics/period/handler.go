package period

import (
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/yusing/go-proxy/internal/api/v1/utils"
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
	period := r.URL.Query().Get("period")
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
		aggregated := p.aggregator(rangeData...)
		utils.RespondJSON(w, r, aggregated)
	} else {
		utils.RespondJSON(w, r, rangeData)
	}
}

func (p *Poller[T, AggregateT]) ServeWS(allowedDomains []string, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	period := query.Get("period")
	intervalStr := query.Get("interval")
	interval, err := time.ParseDuration(intervalStr)

	minInterval := p.interval()
	if err != nil || interval < minInterval {
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
				return wsjson.Write(r.Context(), conn, p.aggregator(p.Get(periodFilter)...))
			})
		} else {
			utils.PeriodicWS(allowedDomains, w, r, interval, func(conn *websocket.Conn) error {
				return wsjson.Write(r.Context(), conn, p.Get(periodFilter))
			})
		}
	}
}
