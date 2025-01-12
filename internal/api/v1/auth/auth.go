package auth

import (
	"bytes"
	"encoding/json"
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

var (
	ErrInvalidUsername = E.New("invalid username")
	ErrInvalidPassword = E.New("invalid password")
)

func validatePassword(cred *Credentials) error {
	if cred.Username != common.APIUser {
		return ErrInvalidUsername.Subject(cred.Username)
	}
	if !bytes.Equal(common.HashPassword(cred.Password), common.APIPasswordHash) {
		return ErrInvalidPassword.Subject(cred.Password)
	}
	return nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		U.HandleErr(w, r, err, http.StatusBadRequest)
		return
	}
	if err := validatePassword(&creds); err != nil {
		U.HandleErr(w, r, err, http.StatusUnauthorized)
		return
	}
	if err := setAuthenticatedCookie(w, creds.Username); err != nil {
		U.HandleErr(w, r, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func AuthMethodHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case common.APIJWTSecret == nil:
		U.WriteBody(w, []byte("skip"))
	case common.OIDCIssuerURL != "":
		U.WriteBody(w, []byte("oidc"))
	case common.APIPasswordHash != nil:
		U.WriteBody(w, []byte("password"))
	default:
		U.WriteBody(w, []byte("skip"))
	}
	w.WriteHeader(http.StatusOK)
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
		Name:     "token",
		Value:    tokenStr,
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	return nil
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	w.Header().Set("location", "/login")
	w.WriteHeader(http.StatusTemporaryRedirect)
}

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

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	if common.IsDebugSkipAuth || common.APIJWTSecret == nil {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if checkToken(w, r) {
			next(w, r)
		}
	}
}

func checkToken(w http.ResponseWriter, r *http.Request) (ok bool) {
	tokenCookie, err := r.Cookie("token")
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
