package period

import (
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/yusing/go-proxy/internal/api/v1/utils"
	metricsutils "github.com/yusing/go-proxy/internal/metrics/utils"
	"github.com/yusing/go-proxy/internal/net/http/httpheaders"
)

// ServeHTTP serves the data for the given period.
//
// If the period is not specified, it serves the last result.
//
// If the period is specified, it serves the data for the given period.
//
// If the period is invalid, it returns a 400 error.
//
// If the data is not found, it returns a 204 error.
//
// If the request is a websocket request, it serves the data for the given period for every interval.
func (p *Poller[T, AggregateT]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if httpheaders.IsWebsocket(r.Header) {
		interval := metricsutils.QueryDuration(query, "interval", 0)

		minInterval := 1 * time.Second
		if interval == 0 {
			interval = p.interval()
		}
		if interval < minInterval {
			interval = minInterval
		}
		utils.PeriodicWS(w, r, interval, func(conn *websocket.Conn) error {
			data, err := p.getRespData(r)
			if err != nil {
				return err
			}
			if data == nil {
				return nil
			}
			return wsjson.Write(r.Context(), conn, data)
		})
	} else {
		data, err := p.getRespData(r)
		if err != nil {
			utils.HandleErr(w, r, err)
			return
		}
		if data == nil {
			http.Error(w, "no data", http.StatusNoContent)
			return
		}
		utils.RespondJSON(w, r, data)
	}
}

func (p *Poller[T, AggregateT]) getRespData(r *http.Request) (any, error) {
	query := r.URL.Query()
	period := query.Get("period")
	if period == "" {
		return p.GetLastResult(), nil
	}
	periodFilter := Filter(period)
	if !periodFilter.IsValid() {
		return nil, errors.New("invalid period")
	}
	rangeData := p.Get(periodFilter)
	if p.aggregator != nil {
		total, aggregated := p.aggregator(rangeData, query)
		return map[string]any{
			"total": total,
			"data":  aggregated,
		}, nil
	} else {
		return rangeData, nil
	}
}
