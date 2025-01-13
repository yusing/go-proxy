package auth

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	Credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	Claims struct {
		Username string `json:"username"`
		jwt.RegisteredClaims
	}
)

// init sets up authentication providers.
func init() {
	if !IsEnabled() {
		logging.Warn().Msg("authentication is disabled, please set API_JWT_SECRET or OIDC_* to enable authentication")
		return
	}
	// Initialize OIDC if configured.
	if common.OIDCIssuerURL != "" {
		if err := initOIDC(
			common.OIDCIssuerURL,
			common.OIDCClientID,
			common.OIDCClientSecret,
			common.OIDCRedirectURL,
		); err != nil {
			logging.Fatal().Err(err).Msg("failed to initialize OIDC provider")
		}
	}
}

func IsEnabled() bool {
	return common.APIJWTSecret != nil || IsOIDCEnabled()
}

func IsOIDCEnabled() bool {
	return common.OIDCIssuerURL != ""
}

// cookieFQDN returns the fully qualified domain name of the request host
// with subdomain stripped.
//
// If the request host does not have a subdomain,
// an empty string is returned
//
//	"abc.example.com" -> "example.com"
//	"example.com" -> ""
func cookieFQDN(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	parts := strutils.SplitRune(host, '.')
	if len(parts) < 2 {
		return ""
	}
	parts[0] = ""
	return strutils.JoinRune(parts, '.')
}

// APIAuthRedirectHandler handles API redirect to login page or OIDC login base on configuration.
func APIAuthRedirectHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case apiOAuth != nil:
		apiOAuth.RedirectOIDC(w, r)
		return
	case common.APIJWTSecret != nil:
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	default:
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func setAuthenticatedCookie(w http.ResponseWriter, r *http.Request, username string) error {
	expiresAt := time.Now().Add(common.APIJWTTokenTTL)
	claim := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claim)
	tokenStr, err := token.SignedString(common.APIJWTSecret)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieToken,
		Value:    tokenStr,
		Expires:  expiresAt,
		Domain:   cookieFQDN(r),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	return nil
}

// LogoutHandler clear authentication cookie and redirect to login page.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieToken,
		MaxAge:   -1,
		Domain:   cookieFQDN(r),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	APIAuthRedirectHandler(w, r)
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	if IsEnabled() {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := CheckToken(w, r); err != nil {
				U.RespondError(w, err, http.StatusUnauthorized)
			} else {
				next(w, r)
			}
		}
	}
	return next
}

func CheckToken(w http.ResponseWriter, r *http.Request) error {
	tokenCookie, err := r.Cookie(CookieToken)
	if err != nil {
		return E.New("missing token")
	}
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenCookie.Value, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return common.APIJWTSecret, nil
	})
	if err != nil {
		return err
	}
	switch {
	case !token.Valid:
		return E.New("invalid token")
	case claims.Username != common.APIUser:
		return E.New("username mismatch").Subject(claims.Username)
	case claims.ExpiresAt.Before(time.Now()):
		return E.Errorf("token expired on %s", strutils.FormatTime(claims.ExpiresAt.Time))
	}

	return nil
}
