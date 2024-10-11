package api

import (
	"fmt"
	"net/http"

	v1 "github.com/yusing/go-proxy/internal/api/v1"
	"github.com/yusing/go-proxy/internal/api/v1/errorpage"
	. "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
)

type ServeMux struct{ *http.ServeMux }

func NewServeMux() ServeMux {
	return ServeMux{http.NewServeMux()}
}

func (mux ServeMux) HandleFunc(method, endpoint string, handler http.HandlerFunc) {
	mux.ServeMux.HandleFunc(fmt.Sprintf("%s %s", method, endpoint), checkHost(handler))
}

func NewHandler(cfg *config.Config) http.Handler {
	mux := NewServeMux()
	mux.HandleFunc("GET", "/v1", v1.Index)
	mux.HandleFunc("GET", "/v1/version", v1.GetVersion)
	mux.HandleFunc("GET", "/v1/checkhealth", wrap(cfg, v1.CheckHealth))
	mux.HandleFunc("HEAD", "/v1/checkhealth", wrap(cfg, v1.CheckHealth))
	mux.HandleFunc("POST", "/v1/reload", wrap(cfg, v1.Reload))
	mux.HandleFunc("GET", "/v1/list", wrap(cfg, v1.List))
	mux.HandleFunc("GET", "/v1/list/{what}", wrap(cfg, v1.List))
	mux.HandleFunc("GET", "/v1/file", v1.GetFileContent)
	mux.HandleFunc("GET", "/v1/file/{filename...}", v1.GetFileContent)
	mux.HandleFunc("POST", "/v1/file/{filename...}", v1.SetFileContent)
	mux.HandleFunc("PUT", "/v1/file/{filename...}", v1.SetFileContent)
	mux.HandleFunc("GET", "/v1/stats", wrap(cfg, v1.Stats))
	mux.HandleFunc("GET", "/v1/stats/ws", wrap(cfg, v1.StatsWS))
	mux.HandleFunc("GET", "/v1/error_page", errorpage.GetHandleFunc())
	return mux
}

// allow only requests to API server with host matching common.APIHTTPAddr.
func checkHost(f http.HandlerFunc) http.HandlerFunc {
	if common.IsDebug {
		return f
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Host != common.APIHTTPAddr {
			Logger.Warnf("invalid request to API server with host: %s, expect %s", r.Host, common.APIHTTPAddr)
			http.Error(w, "invalid request", http.StatusForbidden)
			return
		}
		f(w, r)
	}
}

func wrap(cfg *config.Config, f func(cfg *config.Config, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(cfg, w, r)
	}
}
