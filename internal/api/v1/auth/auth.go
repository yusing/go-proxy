package auth

import (
	"net/http"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/net/gphttp"
)

var defaultAuth Provider

// Initialize sets up authentication providers.
func Initialize() error {
	if !IsEnabled() {
		logging.Warn().Msg("authentication is disabled, please set API_JWT_SECRET or OIDC_* to enable authentication")
		return nil
	}

	var err error
	// Initialize OIDC if configured.
	if common.OIDCIssuerURL != "" {
		defaultAuth, err = NewOIDCProviderFromEnv()
	} else {
		defaultAuth, err = NewUserPassAuthFromEnv()
	}

	return err
}

func GetDefaultAuth() Provider {
	return defaultAuth
}

func IsEnabled() bool {
	return common.APIJWTSecret != nil || IsOIDCEnabled()
}

func IsOIDCEnabled() bool {
	return common.OIDCIssuerURL != ""
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	if IsEnabled() {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := defaultAuth.CheckToken(r); err != nil {
				gphttp.ClientError(w, err, http.StatusUnauthorized)
			} else {
				next(w, r)
			}
		}
	}
	return next
}
