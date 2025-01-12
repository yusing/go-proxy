package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/yusing/go-proxy/internal/common"
	"golang.org/x/oauth2"
)

// setupMockOIDC configures mock OIDC provider for testing.
func setupMockOIDC(t *testing.T) {
	t.Helper()

	oauthConfig = &oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://mock-provider/auth",
			TokenURL: "http://mock-provider/token",
		},
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}
}

func cleanup() {
	oauthConfig = nil
	oidcProvider = nil
	oidcVerifier = nil
}

func TestOIDCLoginHandler(t *testing.T) {
	// Setup
	common.APIJWTSecret = []byte("test-secret")
	common.IsTest = true
	t.Cleanup(func() {
		cleanup()
		common.IsTest = false
	})
	setupMockOIDC(t)

	tests := []struct {
		name           string
		configureOAuth bool
		wantStatus     int
		wantRedirect   bool
	}{
		{
			name:           "Success - Redirects to provider",
			configureOAuth: true,
			wantStatus:     http.StatusTemporaryRedirect,
			wantRedirect:   true,
		},
		{
			name:           "Failure - OIDC not configured",
			configureOAuth: false,
			wantStatus:     http.StatusNotImplemented,
			wantRedirect:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.configureOAuth {
				oauthConfig = nil
			}

			req := httptest.NewRequest(http.MethodGet, "/login/oidc", nil)
			w := httptest.NewRecorder()

			OIDCLoginHandler(w, req)

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
	common.IsTest = true
	t.Cleanup(func() {
		cleanup()
		common.IsTest = false
	})
	tests := []struct {
		name           string
		configureOAuth bool
		state          string
		code           string
		setupMocks     func()
		wantStatus     int
	}{
		{
			name:           "Success - Valid callback",
			configureOAuth: true,
			state:          "valid-state",
			code:           "valid-code",
			setupMocks: func() {
				setupMockOIDC(t)
			},
			wantStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "Failure - OIDC not configured",
			configureOAuth: false,
			wantStatus:     http.StatusNotImplemented,
		},
		{
			name:           "Failure - Missing state",
			configureOAuth: true,
			code:           "valid-code",
			setupMocks: func() {
				setupMockOIDC(t)
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			if !tt.configureOAuth {
				oauthConfig = nil
			}

			req := httptest.NewRequest(http.MethodGet, "/auth/callback?code="+tt.code+"&state="+tt.state, nil)
			if tt.state != "" {
				req.AddCookie(&http.Cookie{
					Name:  "oauth_state",
					Value: tt.state,
				})
			}
			w := httptest.NewRecorder()

			OIDCCallbackHandler(w, req)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("OIDCCallbackHandler() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusTemporaryRedirect {
				cookie := w.Header().Get("Set-Cookie")
				if cookie == "" {
					t.Error("OIDCCallbackHandler() missing token cookie")
				}
			}
		})
	}
}

func TestInitOIDC(t *testing.T) {
	common.IsTest = true
	t.Cleanup(func() {
		common.IsTest = false
	})
	tests := []struct {
		name         string
		issuerURL    string
		clientID     string
		clientSecret string
		redirectURL  string
		wantErr      bool
	}{
		{
			name:         "Success - Empty configuration",
			issuerURL:    "",
			clientID:     "",
			clientSecret: "",
			redirectURL:  "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(cleanup)
			err := InitOIDC(tt.issuerURL, tt.clientID, tt.clientSecret, tt.redirectURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitOIDC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
