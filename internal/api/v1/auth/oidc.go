package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	"golang.org/x/oauth2"
)

type OIDCProvider struct {
	oauthConfig  *oauth2.Config
	oidcProvider *oidc.Provider
	oidcVerifier *oidc.IDTokenVerifier
	allowedUsers []string
	isMiddleware bool
}

const CookieOauthState = "godoxy_oidc_state"

const (
	OIDCMiddlewareCallbackPath = "/auth/callback"
	OIDCLogoutPath             = "/auth/logout"
)

func NewOIDCProvider(issuerURL, clientID, clientSecret, redirectURL string, allowedUsers []string) (*OIDCProvider, error) {
	if len(allowedUsers) == 0 {
		return nil, errors.New("OIDC allowed users must not be empty")
	}

	provider, err := oidc.NewProvider(context.Background(), issuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	return &OIDCProvider{
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       strutils.CommaSeperatedList(common.OIDCScopes),
		},
		oidcProvider: provider,
		oidcVerifier: provider.Verifier(&oidc.Config{
			ClientID: clientID,
		}),
		allowedUsers: allowedUsers,
	}, nil
}

// NewOIDCProviderFromEnv creates a new OIDCProvider from environment variables.
func NewOIDCProviderFromEnv() (*OIDCProvider, error) {
	return NewOIDCProvider(
		common.OIDCIssuerURL,
		common.OIDCClientID,
		common.OIDCClientSecret,
		common.OIDCRedirectURL,
		common.OIDCAllowedUsers,
	)
}

func (auth *OIDCProvider) TokenCookieName() string {
	return "godoxy_oidc_token"
}

func (auth *OIDCProvider) SetIsMiddleware(enabled bool) {
	auth.isMiddleware = enabled
	if auth.isMiddleware {
		auth.oauthConfig.RedirectURL = OIDCMiddlewareCallbackPath
	}
}

func (auth *OIDCProvider) SetAllowedUsers(users []string) {
	auth.allowedUsers = users
}

func (auth *OIDCProvider) CheckToken(r *http.Request) error {
	token, err := r.Cookie(auth.TokenCookieName())
	if err != nil {
		return ErrMissingToken
	}

	// checks for Expiry, Audience == ClientID, Issuer, etc.
	idToken, err := auth.oidcVerifier.Verify(r.Context(), token.Value)
	if err != nil {
		return fmt.Errorf("failed to verify ID token: %w", err)
	}

	if len(idToken.Audience) == 0 {
		return ErrInvalidToken
	}

	var claims struct {
		Email    string `json:"email"`
		Username string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return fmt.Errorf("failed to parse claims: %w", err)
	}

	if !slices.Contains(auth.allowedUsers, claims.Username) {
		return ErrUserNotAllowed.Subject(claims.Username)
	}
	return nil
}

// generateState generates a random string for OIDC state.
const oidcStateLength = 32

func generateState() (string, error) {
	b := make([]byte, oidcStateLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:oidcStateLength], nil
}

// RedirectOIDC initiates the OIDC login flow.
func (auth *OIDCProvider) RedirectLoginPage(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		U.HandleErr(w, r, err, http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieOauthState,
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
		Path:     "/",
	})

	redirURL := auth.oauthConfig.AuthCodeURL(state)
	if auth.isMiddleware {
		u, err := r.URL.Parse(redirURL)
		if err != nil {
			U.HandleErr(w, r, err, http.StatusInternalServerError)
			return
		}
		q := u.Query()
		q.Set("redirect_uri", "https://"+r.Host+q.Get("redirect_uri"))
		u.RawQuery = q.Encode()
		redirURL = u.String()
	}
	http.Redirect(w, r, redirURL, http.StatusTemporaryRedirect)
}

// OIDCCallbackHandler handles the OIDC callback.
func (auth *OIDCProvider) LoginCallbackHandler(w http.ResponseWriter, r *http.Request) {
	// For testing purposes, skip provider verification
	if common.IsTest {
		auth.handleTestCallback(w, r)
		return
	}

	state, err := r.Cookie(CookieOauthState)
	if err != nil {
		U.HandleErr(w, r, E.New("missing state cookie"), http.StatusBadRequest)
		return
	}

	query := r.URL.Query()
	if query.Get("state") != state.Value {
		U.HandleErr(w, r, E.New("invalid oauth state"), http.StatusBadRequest)
		return
	}

	code := query.Get("code")
	oauth2Token, err := auth.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		U.HandleErr(w, r, fmt.Errorf("failed to exchange token: %w", err), http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		U.HandleErr(w, r, E.New("missing id_token"), http.StatusInternalServerError)
		return
	}

	idToken, err := auth.oidcVerifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		U.HandleErr(w, r, fmt.Errorf("failed to verify ID token: %w", err), http.StatusInternalServerError)
		return
	}

	setTokenCookie(w, r, auth.TokenCookieName(), rawIDToken, time.Until(idToken.Expiry))

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// handleTestCallback handles OIDC callback in test environment.
func (auth *OIDCProvider) handleTestCallback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie(CookieOauthState)
	if err != nil {
		U.HandleErr(w, r, E.New("missing state cookie"), http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != state.Value {
		U.HandleErr(w, r, E.New("invalid oauth state"), http.StatusBadRequest)
		return
	}

	// Create test JWT token
	setTokenCookie(w, r, auth.TokenCookieName(), "test", time.Hour)

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
