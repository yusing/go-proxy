package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"golang.org/x/oauth2"
)

var (
	oauthConfig  *oauth2.Config
	oidcProvider *oidc.Provider
	oidcVerifier *oidc.IDTokenVerifier
)

// InitOIDC initializes the OIDC provider
func InitOIDC(issuerURL, clientID, clientSecret, redirectURL string) error {
	if issuerURL == "" {
		return nil // OIDC not configured
	}

	provider, err := oidc.NewProvider(context.Background(), issuerURL)
	if err != nil {
		return fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	oidcProvider = provider
	oidcVerifier = provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return nil
}

// OIDCLoginHandler initiates the OIDC login flow
func OIDCLoginHandler(w http.ResponseWriter, r *http.Request) {
	if oauthConfig == nil {
		U.HandleErr(w, r, E.New("OIDC not configured"), http.StatusNotImplemented)
		return
	}

	state := common.GenerateRandomString(32)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		Path:     "/",
	})

	url := oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// OIDCCallbackHandler handles the OIDC callback
func OIDCCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if oauthConfig == nil {
		U.HandleErr(w, r, E.New("OIDC not configured"), http.StatusNotImplemented)
		return
	}

	// For testing purposes, skip provider verification
	if common.IsTest {
		handleTestCallback(w, r)
		return
	}

	if oidcProvider == nil {
		U.HandleErr(w, r, E.New("OIDC not configured"), http.StatusNotImplemented)
		return
	}

	state, err := r.Cookie("oauth_state")
	if err != nil {
		U.HandleErr(w, r, E.New("missing state cookie"), http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != state.Value {
		U.HandleErr(w, r, E.New("invalid oauth state"), http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	oauth2Token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		U.HandleErr(w, r, fmt.Errorf("failed to exchange token: %w", err), http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		U.HandleErr(w, r, E.New("missing id_token"), http.StatusInternalServerError)
		return
	}

	idToken, err := oidcVerifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		U.HandleErr(w, r, fmt.Errorf("failed to verify ID token: %w", err), http.StatusInternalServerError)
		return
	}

	var claims struct {
		Email    string `json:"email"`
		Username string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		U.HandleErr(w, r, fmt.Errorf("failed to parse claims: %w", err), http.StatusInternalServerError)
		return
	}

	if err := setAuthenticatedCookie(w, r, claims.Username); err != nil {
		U.HandleErr(w, r, err, http.StatusInternalServerError)
		return
	}

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// handleTestCallback handles OIDC callback in test environment
func handleTestCallback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie("oauth_state")
	if err != nil {
		U.HandleErr(w, r, E.New("missing state cookie"), http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != state.Value {
		U.HandleErr(w, r, E.New("invalid oauth state"), http.StatusBadRequest)
		return
	}

	// Create test JWT token
	expiresAt := time.Now().Add(common.APIJWTTokenTTL)
	jwtClaims := &Claims{
		Username: "test-user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwtClaims)
	tokenStr, err := token.SignedString(common.APIJWTSecret)
	if err != nil {
		U.HandleErr(w, r, err, http.StatusInternalServerError)
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

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
