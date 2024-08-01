package api

import (
	"net/http"

	v1 "github.com/yusing/go-proxy/api/v1"
	"github.com/yusing/go-proxy/config"
)

func NewHandler(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1", v1.Index)
	mux.HandleFunc("GET /v1/checkhealth", wrap(cfg, v1.CheckHealth))
	mux.HandleFunc("HEAD /v1/checkhealth", wrap(cfg, v1.CheckHealth))
	mux.HandleFunc("POST /v1/reload", wrap(cfg, v1.Reload))
	mux.HandleFunc("GET /v1/list", wrap(cfg, v1.List))
	mux.HandleFunc("GET /v1/list/{what}", wrap(cfg, v1.List))
	mux.HandleFunc("GET /v1/file", v1.GetFileContent)
	mux.HandleFunc("GET /v1/file/{filename}", v1.GetFileContent)
	mux.HandleFunc("PUT /v1/file/{filename}", v1.SetFileContent)
	mux.HandleFunc("GET /v1/stats", wrap(cfg, v1.Stats))
	return mux
}

func wrap(cfg *config.Config, f func(cfg *config.Config, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(cfg, w, r)
	}
}
