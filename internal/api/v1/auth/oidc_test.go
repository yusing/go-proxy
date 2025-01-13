package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"golang.org/x/oauth2"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

// setupMockOIDC configures mock OIDC provider for testing.
func setupMockOIDC(t *testing.T) {
	t.Helper()

	provider := (&oidc.ProviderConfig{}).NewProvider(context.TODO())
	defaultAuth = &OIDCProvider{
		oauthConfig: &oauth2.Config{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "http://mock-provider/auth",
				TokenURL: "http://mock-provider/token",
			},
			Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
		},
		oidcProvider: provider,
		oidcVerifier: provider.Verifier(&oidc.Config{
			ClientID: "test-client",
		}),
		allowedUsers: []string{"test-user"},
	}
}

func cleanup() {
	defaultAuth = nil
}

func TestOIDCLoginHandler(t *testing.T) {
	// Setup
	common.APIJWTSecret = []byte("test-secret")
	t.Cleanup(cleanup)
	setupMockOIDC(t)

	tests := []struct {
		name         string
		wantStatus   int
		wantRedirect bool
	}{
		{
			name:         "Success - Redirects to provider",
			wantStatus:   http.StatusTemporaryRedirect,
			wantRedirect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/auth/redirect", nil)
			w := httptest.NewRecorder()

			defaultAuth.RedirectLoginPage(w, req)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("OIDCLoginHandler() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantRedirect {
				if loc := w.Header().Get("Location"); loc == "" {
					t.Error("OIDCLoginHandler() missing redirect location")
				}

				cookie := w.Header().Get("Set-Cookie")
				if cookie == "" {
					t.Error("OIDCLoginHandler() missing state cookie")
				}
			}
		})
	}
}

func TestOIDCCallbackHandler(t *testing.T) {
	// Setup
	common.APIJWTSecret = []byte("test-secret")
	t.Cleanup(cleanup)
	tests := []struct {
		name       string
		state      string
		code       string
		setupMocks bool
		wantStatus int
	}{
		{
			name:       "Success - Valid callback",
			state:      "valid-state",
			code:       "valid-code",
			setupMocks: true,
			wantStatus: http.StatusTemporaryRedirect,
		},
		{
			name:       "Failure - Missing state",
			code:       "valid-code",
			setupMocks: true,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks {
				setupMockOIDC(t)
			}

			req := httptest.NewRequest(http.MethodGet, "/auth/callback?code="+tt.code+"&state="+tt.state, nil)
			if tt.state != "" {
				req.AddCookie(&http.Cookie{
					Name:  CookieOauthState,
					Value: tt.state,
				})
			}
			w := httptest.NewRecorder()

			defaultAuth.LoginCallbackHandler(w, req)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("OIDCCallbackHandler() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusTemporaryRedirect {
				setCookie := E.Must(http.ParseSetCookie(w.Header().Get("Set-Cookie")))
				ExpectEqual(t, setCookie.Name, defaultAuth.TokenCookieName())
				ExpectTrue(t, setCookie.Value != "")
				ExpectEqual(t, setCookie.Path, "/")
				ExpectEqual(t, setCookie.SameSite, http.SameSiteLaxMode)
				ExpectEqual(t, setCookie.HttpOnly, true)
			}
		})
	}
}

func TestInitOIDC(t *testing.T) {
	tests := []struct {
		name         string
		issuerURL    string
		clientID     string
		clientSecret string
		redirectURL  string
		allowedUsers []string
		wantErr      bool
	}{
		{
			name:         "Fail - Empty configuration",
			issuerURL:    "",
			clientID:     "",
			clientSecret: "",
			redirectURL:  "",
			allowedUsers: nil,
			wantErr:      true,
		},
		// {
		// 	name:         "Success - Valid configuration",
		// 	issuerURL:    "https://example.com",
		// 	clientID:     "client_id",
		// 	clientSecret: "client_secret",
		// 	redirectURL:  "https://example.com/callback",
		// 	allowedUsers: []string{"user1", "user2"},
		// 	wantErr:      false,
		// },
		{
			name:         "Fail - No allowed users",
			issuerURL:    "https://example.com",
			clientID:     "client_id",
			clientSecret: "client_secret",
			redirectURL:  "https://example.com/callback",
			allowedUsers: []string{},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(cleanup)
			_, err := NewOIDCProvider(tt.issuerURL, tt.clientID, tt.clientSecret, tt.redirectURL, tt.allowedUsers)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitOIDC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
