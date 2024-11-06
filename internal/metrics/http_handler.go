package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}
