package auth

import (
	"context"
	"fmt"
	"net/http"

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
	overrideHost bool
}

var (
	apiOAuth               *OIDCProvider
	APIOIDCCallbackHandler http.HandlerFunc
)

// initOIDC initializes the OIDC provider.
func initOIDC(issuerURL, clientID, clientSecret, redirectURL string) (err error) {
	if issuerURL == "" {
		return nil // OIDC not configured
	}

	apiOAuth, err = NewOIDCProvider(issuerURL, clientID, clientSecret, redirectURL)
	APIOIDCCallbackHandler = apiOAuth.OIDCCallbackHandler
	return
}

func NewOIDCProvider(issuerURL, clientID, clientSecret, redirectURL string) (*OIDCProvider, error) {
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
	}, nil
}

func NewOIDCProviderFromEnv(redirectURL string) (*OIDCProvider, error) {
	return NewOIDCProvider(
		common.OIDCIssuerURL,
		common.OIDCClientID,
		common.OIDCClientSecret,
		redirectURL,
	)
}

func (provider *OIDCProvider) SetOverrideHostEnabled(enabled bool) {
	provider.overrideHost = enabled
}

// RedirectOIDC initiates the OIDC login flow.
func (provider *OIDCProvider) RedirectOIDC(w http.ResponseWriter, r *http.Request) {
	if provider == nil {
		U.HandleErr(w, r, E.New("OIDC not configured"), http.StatusNotImplemented)
		return
	}

	state := common.GenerateRandomString(32)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieOauthState,
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
		Path:     "/",
	})

	redirURL := provider.oauthConfig.AuthCodeURL(state)
	if provider.overrideHost {
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
func (provider *OIDCProvider) OIDCCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if provider == nil {
		U.HandleErr(w, r, E.New("OIDC not configured"), http.StatusNotImplemented)
		return
	}

	// For testing purposes, skip provider verification
	if common.IsTest {
		handleTestCallback(w, r)
		return
	}

	if provider.oidcProvider == nil {
		U.HandleErr(w, r, E.New("OIDC not configured"), http.StatusNotImplemented)
		return
	}

	state, err := r.Cookie(CookieOauthState)
	if err != nil {
		U.HandleErr(w, r, E.New("missing state cookie"), http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != state.Value {
		U.HandleErr(w, r, E.New("invalid oauth state"), http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	oauth2Token, err := provider.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		U.HandleErr(w, r, fmt.Errorf("failed to exchange token: %w", err), http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		U.HandleErr(w, r, E.New("missing id_token"), http.StatusInternalServerError)
		return
	}

	idToken, err := provider.oidcVerifier.Verify(r.Context(), rawIDToken)
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

// handleTestCallback handles OIDC callback in test environment.
func handleTestCallback(w http.ResponseWriter, r *http.Request) {
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
	if err := setAuthenticatedCookie(w, r, "test-user"); err != nil {
		U.HandleErr(w, r, err, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
