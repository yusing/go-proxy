package auth

import (
	"net/http"
)

type Provider interface {
	TokenCookieName() string
	CheckToken(w http.ResponseWriter, r *http.Request) error
	RedirectLoginPage(w http.ResponseWriter, r *http.Request)
	LoginCallbackHandler(w http.ResponseWriter, r *http.Request)
}
