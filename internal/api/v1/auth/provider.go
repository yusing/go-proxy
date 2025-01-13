package auth

import (
	"net/http"
)

type Provider interface {
	TokenCookieName() string
	CheckToken(r *http.Request) error
	RedirectLoginPage(w http.ResponseWriter, r *http.Request)
	LoginCallbackHandler(w http.ResponseWriter, r *http.Request)
}
