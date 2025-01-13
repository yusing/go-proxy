package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
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

// Initialize sets up authentication providers.
func Initialize() error {
	// Initialize OIDC if configured.
	if common.OIDCIssuerURL != "" {
		return InitOIDC(
			common.OIDCIssuerURL,
			common.OIDCClientID,
			common.OIDCClientSecret,
			common.OIDCRedirectURL,
		)
	}
	return nil
}

func IsEnabled() bool {
	return common.APIJWTSecret != nil || common.OIDCIssuerURL != ""
}

// AuthRedirectHandler handles redirect to login page or OIDC login base on configuration.
func AuthRedirectHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case oauthConfig != nil:
		RedirectOIDC(w, r)
		return
	case common.APIJWTSecret != nil:
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	default:
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func setAuthenticatedCookie(w http.ResponseWriter, username string) error {
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
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	return nil
}

// LogoutHandler clear authentication cookie and redirect to login page.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieToken,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	AuthRedirectHandler(w, r)
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	if IsEnabled() {
		return func(w http.ResponseWriter, r *http.Request) {
			if checkToken(w, r) {
				next(w, r)
			}
		}
	}
	return next
}

func checkToken(w http.ResponseWriter, r *http.Request) (ok bool) {
	tokenCookie, err := r.Cookie(CookieToken)
	if err != nil {
		U.RespondError(w, E.New("missing token"), http.StatusUnauthorized)
		return false
	}
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenCookie.Value, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return common.APIJWTSecret, nil
	})

	switch {
	case err != nil:
		break
	case !token.Valid:
		err = E.New("invalid token")
	case claims.Username != common.APIUser:
		err = E.New("username mismatch").Subject(claims.Username)
	case claims.ExpiresAt.Before(time.Now()):
		err = E.Errorf("token expired on %s", strutils.FormatTime(claims.ExpiresAt.Time))
	}

	if err != nil {
		U.RespondError(w, err, http.StatusForbidden)
		return false
	}

	return true
}
