package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "github.com/yusing/go-proxy/internal/api/v1"
	"github.com/yusing/go-proxy/internal/api/v1/auth"
	"github.com/yusing/go-proxy/internal/api/v1/favicon"
	"github.com/yusing/go-proxy/internal/common"
	config "github.com/yusing/go-proxy/internal/config/types"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/logging/memlogger"
	"github.com/yusing/go-proxy/internal/metrics/uptime"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type ServeMux struct{ *http.ServeMux }

func (mux ServeMux) HandleFunc(methods, endpoint string, handler http.HandlerFunc) {
	for _, m := range strutils.CommaSeperatedList(methods) {
		mux.ServeMux.HandleFunc(m+" "+endpoint, handler)
	}
}

func NewHandler(cfg config.ConfigInstance) http.Handler {
	mux := ServeMux{http.NewServeMux()}
	mux.HandleFunc("GET", "/v1", v1.Index)
	mux.HandleFunc("GET", "/v1/version", v1.GetVersion)
	mux.HandleFunc("POST", "/v1/reload", useCfg(cfg, v1.Reload))
	mux.HandleFunc("GET", "/v1/list", auth.RequireAuth(useCfg(cfg, v1.List)))
	mux.HandleFunc("GET", "/v1/list/{what}", auth.RequireAuth(useCfg(cfg, v1.List)))
	mux.HandleFunc("GET", "/v1/list/{what}/{which}", auth.RequireAuth(useCfg(cfg, v1.List)))
	mux.HandleFunc("GET", "/v1/file/{type}/{filename}", auth.RequireAuth(v1.GetFileContent))
	mux.HandleFunc("POST,PUT", "/v1/file/{type}/{filename}", auth.RequireAuth(v1.SetFileContent))
	mux.HandleFunc("POST", "/v1/file/validate/{type}", auth.RequireAuth(v1.ValidateFile))
	mux.HandleFunc("GET", "/v1/stats", useCfg(cfg, v1.Stats))
	mux.HandleFunc("GET", "/v1/stats/ws", useCfg(cfg, v1.StatsWS))
	mux.HandleFunc("GET", "/v1/health/ws", auth.RequireAuth(useCfg(cfg, v1.HealthWS)))
	mux.HandleFunc("GET", "/v1/logs/ws", auth.RequireAuth(memlogger.LogsWS(cfg.Value().MatchDomains)))
	mux.HandleFunc("GET", "/v1/favicon", auth.RequireAuth(favicon.GetFavIcon))
	mux.HandleFunc("POST", "/v1/homepage/set", auth.RequireAuth(v1.SetHomePageOverrides))
	mux.HandleFunc("GET", "/v1/agents/ws", auth.RequireAuth(useCfg(cfg, v1.AgentsWS)))
	mux.HandleFunc("GET", "/v1/metrics/system_info", auth.RequireAuth(useCfg(cfg, v1.SystemInfo)))
	mux.HandleFunc("GET", "/v1/metrics/system_info/ws", auth.RequireAuth(useCfg(cfg, v1.SystemInfo)))
	mux.HandleFunc("GET", "/v1/metrics/uptime", auth.RequireAuth(uptime.Poller.ServeHTTP))
	mux.HandleFunc("GET", "/v1/metrics/uptime/ws", auth.RequireAuth(useWS(cfg, uptime.Poller.ServeWS)))

	if common.PrometheusEnabled {
		mux.Handle("GET /v1/metrics", promhttp.Handler())
		logging.Info().Msg("prometheus metrics enabled")
	}

	defaultAuth := auth.GetDefaultAuth()
	if defaultAuth != nil {
		mux.HandleFunc("GET", "/v1/auth/redirect", defaultAuth.RedirectLoginPage)
		mux.HandleFunc("GET", "/v1/auth/check", func(w http.ResponseWriter, r *http.Request) {
			if err := defaultAuth.CheckToken(r); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
		})
		mux.HandleFunc("GET,POST", "/v1/auth/callback", defaultAuth.LoginCallbackHandler)
		mux.HandleFunc("GET,POST", "/v1/auth/logout", defaultAuth.LogoutCallbackHandler)
	} else {
		mux.HandleFunc("GET", "/v1/auth/check", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
	return mux
}

func useCfg(cfg config.ConfigInstance, handler func(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(cfg, w, r)
	}
}

func useWS(cfg config.ConfigInstance, handler func(allowedDomains []string, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(cfg.Value().MatchDomains, w, r)
	}
}
