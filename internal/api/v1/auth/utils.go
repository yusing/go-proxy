package auth

import (
	"net"
	"net/http"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

var (
	ErrMissingToken   = E.New("missing token")
	ErrInvalidToken   = E.New("invalid token")
	ErrUserNotAllowed = E.New("user not allowed")
)

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

func setTokenCookie(w http.ResponseWriter, r *http.Request, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(ttl.Seconds()),
		Domain:   cookieFQDN(r),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
}

func clearTokenCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Domain:   cookieFQDN(r),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
}

func LogoutCallbackHandler(auth Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clearTokenCookie(w, r, auth.TokenCookieName())
		auth.RedirectLoginPage(w, r)
	}
}
