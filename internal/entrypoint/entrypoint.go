package entrypoint

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/http/accesslog"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/net/http/middleware/errorpage"
	"github.com/yusing/go-proxy/internal/route/routes"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/task"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

var findRouteFunc = findRouteAnyDomain

var (
	epMiddleware   *middleware.Middleware
	epMiddlewareMu sync.Mutex

	epAccessLogger   *accesslog.AccessLogger
	epAccessLoggerMu sync.Mutex
)

var ErrNoSuchRoute = errors.New("no such route")

func SetFindRouteDomains(domains []string) {
	if len(domains) == 0 {
		findRouteFunc = findRouteAnyDomain
	} else {
		findRouteFunc = findRouteByDomains(domains)
	}
}

func SetMiddlewares(mws []map[string]any) error {
	epMiddlewareMu.Lock()
	defer epMiddlewareMu.Unlock()

	if len(mws) == 0 {
		epMiddleware = nil
		return nil
	}

	mid, err := middleware.BuildMiddlewareFromChainRaw("entrypoint", mws)
	if err != nil {
		return err
	}
	epMiddleware = mid

	logger.Debug().Msg("entrypoint middleware loaded")
	return nil
}

func SetAccessLogger(parent task.Parent, cfg *accesslog.Config) (err error) {
	epAccessLoggerMu.Lock()
	defer epAccessLoggerMu.Unlock()

	if cfg == nil {
		epAccessLogger = nil
		return
	}

	epAccessLogger, err = accesslog.NewFileAccessLogger(parent, cfg)
	if err != nil {
		return
	}
	logger.Debug().Msg("entrypoint access logger created")
	return
}

func Handler(w http.ResponseWriter, r *http.Request) {
	mux, err := findRouteFunc(r.Host)
	if err == nil {
		if epAccessLogger != nil {
			epMiddlewareMu.Lock()
			if epAccessLogger != nil {
				w = gphttp.NewModifyResponseWriter(w, r, func(resp *http.Response) error {
					epAccessLogger.Log(r, resp)
					return nil
				})
			}
			epMiddlewareMu.Unlock()
		}
		if epMiddleware != nil {
			epMiddlewareMu.Lock()
			if epMiddleware != nil {
				mid := epMiddleware
				epMiddlewareMu.Unlock()
				mid.ServeHTTP(mux.ServeHTTP, w, r)
				return
			}
			epMiddlewareMu.Unlock()
		}
		mux.ServeHTTP(w, r)
		return
	}
	// Why use StatusNotFound instead of StatusBadRequest or StatusBadGateway?
	// On nginx, when route for domain does not exist, it returns StatusBadGateway.
	// Then scraper / scanners will know the subdomain is invalid.
	// With StatusNotFound, they won't know whether it's the path, or the subdomain that is invalid.
	if served := middleware.ServeStaticErrorPageFile(w, r); !served {
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
	hostSplit := strutils.SplitRune(host, '.')
	target := hostSplit[0]

	if r, ok := routes.GetHTTPRouteOrExact(target, host); ok {
		return r, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrNoSuchRoute, target)
}

func findRouteByDomains(domains []string) func(host string) (route.HTTPRoute, error) {
	return func(host string) (route.HTTPRoute, error) {
		for _, domain := range domains {
			if strings.HasSuffix(host, domain) {
				target := strings.TrimSuffix(host, domain)
				if r, ok := routes.GetHTTPRoute(target); ok {
					return r, nil
				}
			}
		}

		// fallback to exact match
		if r, ok := routes.GetHTTPRoute(host); ok {
			return r, nil
		}
		return nil, fmt.Errorf("%w: %s", ErrNoSuchRoute, host)
	}
}
