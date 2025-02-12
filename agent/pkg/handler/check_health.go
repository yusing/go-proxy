package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	apiUtils "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/watcher/health"
	"github.com/yusing/go-proxy/internal/watcher/health/monitor"
)

func CheckHealth(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	scheme := query.Get("scheme")
	if scheme == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var result *health.HealthCheckResult
	var err error
	switch scheme {
	case "fileserver":
		path := query.Get("path")
		if path == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		_, err := os.Stat(path)
		result = &health.HealthCheckResult{Healthy: err == nil}
		if err != nil {
			result.Detail = err.Error()
		}
	case "http", "https": // path is optional
		host := query.Get("host")
		path := query.Get("path")
		if host == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		result, err = monitor.NewHTTPHealthChecker(types.NewURL(&url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   path,
		}), health.DefaultHealthConfig).CheckHealth()
	case "tcp", "udp":
		host := query.Get("host")
		if host == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		hasPort := strings.Contains(host, ":")
		port := query.Get("port")
		if port != "" && !hasPort {
			host = fmt.Sprintf("%s:%s", host, port)
		} else {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		result, err = monitor.NewRawHealthChecker(types.NewURL(&url.URL{
			Scheme: scheme,
			Host:   host,
		}), health.DefaultHealthConfig).CheckHealth()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	apiUtils.RespondJSON(w, r, result)
}
