package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidUsername = E.New("invalid username")
	ErrInvalidPassword = E.New("invalid password")
)

type (
	UserPassAuth struct {
		username string
		pwdHash  []byte
		secret   []byte
		tokenTTL time.Duration
	}
	UserPassClaims struct {
		Username string `json:"username"`
		jwt.RegisteredClaims
	}
)

func NewUserPassAuth(username, password string, secret []byte, tokenTTL time.Duration) (*UserPassAuth, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &UserPassAuth{
		username: username,
		pwdHash:  hash,
		secret:   secret,
		tokenTTL: tokenTTL,
	}, nil
}

func NewUserPassAuthFromEnv() (*UserPassAuth, error) {
	return NewUserPassAuth(
		common.APIUser,
		common.APIPassword,
		common.APIJWTSecret,
		common.APIJWTTokenTTL,
	)
}

func (auth *UserPassAuth) TokenCookieName() string {
	return "godoxy_token"
}

func (auth *UserPassAuth) NewToken() (token string, err error) {
	claim := &UserPassClaims{
		Username: auth.username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(auth.tokenTTL)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS512, claim)
	token, err = tok.SignedString(auth.secret)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (auth *UserPassAuth) CheckToken(r *http.Request) error {
	jwtCookie, err := r.Cookie(auth.TokenCookieName())
	if err != nil {
		return ErrMissingToken
	}
	var claims UserPassClaims
	token, err := jwt.ParseWithClaims(jwtCookie.Value, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return auth.secret, nil
	})
	if err != nil {
		return err
	}
	switch {
	case !token.Valid:
		return ErrInvalidToken
	case claims.Username != auth.username:
		return ErrUserNotAllowed.Subject(claims.Username)
	case claims.ExpiresAt.Before(time.Now()):
		return E.Errorf("token expired on %s", strutils.FormatTime(claims.ExpiresAt.Time))
	}

	return nil
}

func (auth *UserPassAuth) RedirectLoginPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}

func (auth *UserPassAuth) LoginCallbackHandler(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		User string `json:"username"`
		Pass string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		U.HandleErr(w, r, err, http.StatusBadRequest)
		return
	}
	if err := auth.validatePassword(creds.User, creds.Pass); err != nil {
		U.HandleErr(w, r, err, http.StatusUnauthorized)
		return
	}
	token, err := auth.NewToken()
	if err != nil {
		U.HandleErr(w, r, err, http.StatusInternalServerError)
		return
	}
	setTokenCookie(w, r, auth.TokenCookieName(), token, auth.tokenTTL)
	w.WriteHeader(http.StatusOK)
}

func (auth *UserPassAuth) validatePassword(user, pass string) error {
	if user != auth.username {
		return ErrInvalidUsername.Subject(user)
	}
	if err := bcrypt.CompareHashAndPassword(auth.pwdHash, []byte(pass)); err != nil {
		return ErrInvalidPassword.With(err).Subject(pass)
	}
	return nil
}
