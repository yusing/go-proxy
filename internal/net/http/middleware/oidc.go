package middleware

import (
	"net/http"

	"github.com/yusing/go-proxy/internal/api/v1/auth"
	E "github.com/yusing/go-proxy/internal/error"
)

type oidcMiddleware struct {
	oauth   *auth.OIDCProvider
	authMux *http.ServeMux
}

var OIDC = NewMiddleware[oidcMiddleware]()

const (
	OIDCMiddlewareCallbackPath = "/godoxy-auth-oidc/callback"
	OIDCLogoutPath             = "/logout"
)

func (amw *oidcMiddleware) finalize() error {
	if !auth.IsOIDCEnabled() {
		return E.New("OIDC not enabled but Auth middleware is used")
	}
	provider, err := auth.NewOIDCProviderFromEnv(OIDCMiddlewareCallbackPath)
	if err != nil {
		return err
	}
	provider.SetOverrideHostEnabled(true)
	amw.oauth = provider
	amw.authMux = http.NewServeMux()
	amw.authMux.HandleFunc(OIDCMiddlewareCallbackPath, provider.OIDCCallbackHandler)
	amw.authMux.HandleFunc(OIDCLogoutPath, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
	amw.authMux.HandleFunc("/", provider.RedirectOIDC)
	return nil
}

func (amw *oidcMiddleware) before(w http.ResponseWriter, r *http.Request) (proceed bool) {
	if err, _ := auth.CheckToken(w, r); err != nil {
		amw.authMux.ServeHTTP(w, r)
		return false
	}
	if r.URL.Path == OIDCLogoutPath {
		auth.LogoutHandler(w, r)
		return false
	}
	return true
}
