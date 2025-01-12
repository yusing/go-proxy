package api

import (
	"net"
	"net/http"

	v1 "github.com/yusing/go-proxy/internal/api/v1"
	"github.com/yusing/go-proxy/internal/api/v1/auth"
	. "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	config "github.com/yusing/go-proxy/internal/config/types"
)

type ServeMux struct{ *http.ServeMux }

func (mux ServeMux) HandleFunc(method, endpoint string, handler http.HandlerFunc) {
	mux.ServeMux.HandleFunc(method+" "+endpoint, checkHost(handler))
}

func NewHandler(cfg config.ConfigInstance) http.Handler {
	mux := ServeMux{http.NewServeMux()}
	mux.HandleFunc("GET", "/v1", v1.Index)
	mux.HandleFunc("GET", "/v1/version", v1.GetVersion)
	mux.HandleFunc("POST", "/v1/login", auth.LoginHandler)
	mux.HandleFunc("GET", "/v1/logout", auth.LogoutHandler)
	mux.HandleFunc("POST", "/v1/logout", auth.LogoutHandler)
	mux.HandleFunc("POST", "/v1/reload", useCfg(cfg, v1.Reload))
	mux.HandleFunc("GET", "/v1/list", auth.RequireAuth(useCfg(cfg, v1.List)))
	mux.HandleFunc("GET", "/v1/list/{what}", auth.RequireAuth(useCfg(cfg, v1.List)))
	mux.HandleFunc("GET", "/v1/list/{what}/{which}", auth.RequireAuth(useCfg(cfg, v1.List)))
	mux.HandleFunc("GET", "/v1/file/{type}/{filename}", auth.RequireAuth(v1.GetFileContent))
	mux.HandleFunc("POST", "/v1/file/{type}/{filename}", auth.RequireAuth(v1.SetFileContent))
	mux.HandleFunc("PUT", "/v1/file/{type}/{filename}", auth.RequireAuth(v1.SetFileContent))
	mux.HandleFunc("GET", "/v1/schema/{filename...}", v1.GetSchemaFile)
	mux.HandleFunc("GET", "/v1/stats", useCfg(cfg, v1.Stats))
	mux.HandleFunc("GET", "/v1/stats/ws", useCfg(cfg, v1.StatsWS))
	mux.HandleFunc("GET", "/v1/favicon/{alias}", auth.RequireAuth(v1.GetFavIcon))
	return mux
}

func useCfg(cfg config.ConfigInstance, handler func(cfg config.ConfigInstance, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(cfg, w, r)
	}
}

// allow only requests to API server with localhost.
func checkHost(f http.HandlerFunc) http.HandlerFunc {
	if common.IsDebug {
		return f
	}
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host != "127.0.0.1" && host != "localhost" && host != "[::1]" {
			LogWarn(r).Msgf("blocked API request from %s", host)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		LogDebug(r).Interface("headers", r.Header).Msg("API request")
		f(w, r)
	}
}
