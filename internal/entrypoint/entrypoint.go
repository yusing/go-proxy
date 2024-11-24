package entrypoint

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/net/http/middleware/errorpage"
	"github.com/yusing/go-proxy/internal/route/routes"
	route "github.com/yusing/go-proxy/internal/route/types"
)

var findRouteFunc = findRouteAnyDomain

func SetFindRouteDomains(domains []string) {
	if len(domains) == 0 {
		findRouteFunc = findRouteAnyDomain
	} else {
		findRouteFunc = findRouteByDomains(domains)
	}
}

func Handler(w http.ResponseWriter, r *http.Request) {
	mux, err := findRouteFunc(r.Host)
	if err == nil {
		mux.ServeHTTP(w, r)
		return
	}
	// Why use StatusNotFound instead of StatusBadRequest or StatusBadGateway?
	// On nginx, when route for domain does not exist, it returns StatusBadGateway.
	// Then scraper / scanners will know the subdomain is invalid.
	// With StatusNotFound, they won't know whether it's the path, or the subdomain that is invalid.
	if !middleware.ServeStaticErrorPageFile(w, r) {
		logger.Err(err).Str("method", r.Method).Str("url", r.URL.String()).Msg("request")
		errorPage, ok := errorpage.GetErrorPageByStatus(http.StatusNotFound)
		if ok {
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if _, err := w.Write(errorPage); err != nil {
				logger.Err(err).Msg("failed to write error page")
			}
		} else {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
	}
}

func findRouteAnyDomain(host string) (route.HTTPRoute, error) {
	hostSplit := strings.Split(host, ".")
	n := len(hostSplit)
	switch {
	case n == 3:
		host = hostSplit[0]
	case n > 3:
		var builder strings.Builder
		builder.Grow(2*n - 3)
		builder.WriteString(hostSplit[0])
		for _, part := range hostSplit[:n-2] {
			builder.WriteRune('.')
			builder.WriteString(part)
		}
		host = builder.String()
	default:
		return nil, errors.New("missing subdomain in url")
	}
	if r, ok := routes.GetHTTPRoute(host); ok {
		return r, nil
	}
	return nil, fmt.Errorf("no such route: %s", host)
}

func findRouteByDomains(domains []string) func(host string) (route.HTTPRoute, error) {
	return func(host string) (route.HTTPRoute, error) {
		var subdomain string

		for _, domain := range domains {
			if strings.HasSuffix(host, domain) {
				subdomain = strings.TrimSuffix(host, domain)
				break
			}
		}

		if subdomain != "" { // matched
			if r, ok := routes.GetHTTPRoute(subdomain); ok {
				return r, nil
			}
			return nil, fmt.Errorf("no such route: %s", subdomain)
		}
		return nil, fmt.Errorf("%s does not match any base domain", host)
	}
}
