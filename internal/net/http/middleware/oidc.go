package middleware

import (
	"net/http"

	"github.com/yusing/go-proxy/internal/api/v1/auth"
	E "github.com/yusing/go-proxy/internal/error"
)

type oidcMiddleware struct {
	AllowedUsers []string `json:"allowed_users"`

	auth          auth.Provider
	authMux       *http.ServeMux
	logoutHandler http.HandlerFunc
}

var OIDC = NewMiddleware[oidcMiddleware]()

func (amw *oidcMiddleware) finalize() error {
	if !auth.IsOIDCEnabled() {
		return E.New("OIDC not enabled but ODIC middleware is used")
	}
	authProvider, err := auth.NewOIDCProviderFromEnv()
	if err != nil {
		return err
	}

	authProvider.SetIsMiddleware(true)
	if len(amw.AllowedUsers) > 0 {
		authProvider.SetAllowedUsers(amw.AllowedUsers)
	}

	amw.authMux = http.NewServeMux()
	amw.authMux.HandleFunc(auth.OIDCMiddlewareCallbackPath, authProvider.LoginCallbackHandler)
	amw.authMux.HandleFunc(auth.OIDCLogoutPath, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
	amw.authMux.HandleFunc("/", authProvider.RedirectLoginPage)
	amw.logoutHandler = auth.LogoutCallbackHandler(authProvider)
	amw.auth = authProvider
	return nil
}

func (amw *oidcMiddleware) before(w http.ResponseWriter, r *http.Request) (proceed bool) {
	if err := amw.auth.CheckToken(r); err != nil {
		amw.authMux.ServeHTTP(w, r)
		return false
	}
	if r.URL.Path == auth.OIDCLogoutPath {
		amw.logoutHandler(w, r)
		return false
	}
	return true
}
