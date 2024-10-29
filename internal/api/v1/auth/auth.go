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

const tokenExpiration = 24 * time.Hour

const jwtClaimKeyUsername = "username"

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

	expiresAt := time.Now().Add(tokenExpiration)
	claim := &Claims{
		Username: creds.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES512, claim)
	tokenStr, err := token.SignedString(common.APIJWTSecret)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenStr,
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	w.WriteHeader(http.StatusOK)
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	if common.IsDebugSkipAuth {
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
		U.HandleErr(w, r, E.PrependSubject("token", err), http.StatusUnauthorized)
		return false
	}
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenCookie.Value, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
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
		U.HandleErr(w, r, err, http.StatusForbidden)
		return false
	}

	return true
}
