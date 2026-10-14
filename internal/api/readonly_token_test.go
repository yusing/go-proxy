package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/internal/auth"
	"github.com/yusing/godoxy/internal/common"
	"github.com/yusing/godoxy/internal/entrypoint"
	"github.com/yusing/godoxy/internal/route"
	"github.com/yusing/goutils/task"
)

func TestReadOnlyBearerHandlers(t *testing.T) {
	t.Setenv("GODOXY_API_READONLY_TOKEN", "machine-test-credential")
	handler := newAuthenticatedHandler(t)
	parent := task.GetTestTask(t)
	ep := entrypoint.NewEntrypoint(parent, nil)
	entrypoint.SetCtx(parent, ep)
	ep.ExcludedRoutes().Add(&route.Route{Alias: "app"})
	preserveRuleHandlers(t)
	require.NoError(t, RegisterHandlers())
	for name, h := range map[string]http.Handler{"standalone": handler, "embedded": embeddedWebUIHandler(t)} {
		t.Run(name, func(t *testing.T) {
			for _, path := range []string{"/api/v1/route/list", "/api/v1/route/app"} {
				req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(parent.Context())
				req.Header.Set("Authorization", "bEaReR machine-test-credential")
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)
				require.Equal(t, http.StatusOK, rec.Code)
				assert.Contains(t, rec.Body.String(), `"alias":"app"`)
				assert.Empty(t, rec.Result().Cookies())
				assert.NotContains(t, rec.Body.String(), "machine-test-credential")
			}
		})
	}

	// A newly registered endpoint must not inherit token access, even under /route.
	called := false
	handler.GET("/api/v1/route/future", AuthMiddleware(), CSRFMiddleware(), func(c *gin.Context) {
		called = true
		c.Status(http.StatusNoContent)
	})
	session := &http.Cookie{Name: "godoxy_token", Value: issueSessionToken(t)}
	for _, tc := range []struct {
		name, method, path string
		authorization      []string
		upgrade            bool
		status             int
	}{
		{"invalid with session", "GET", "/api/v1/route/list", []string{"Bearer wrong"}, false, 401},
		{"missing credential", "GET", "/api/v1/route/list", []string{"Bearer"}, false, 401},
		{"empty header", "GET", "/api/v1/route/list", []string{""}, false, 401},
		{"unsupported scheme", "GET", "/api/v1/route/list", []string{"Basic machine-test-credential"}, false, 401},
		{"malformed whitespace", "GET", "/api/v1/route/list", []string{"Bearer  machine-test-credential"}, false, 401},
		{"duplicate credentials", "GET", "/api/v1/route/list", []string{"Bearer machine-test-credential", "Bearer wrong"}, false, 401},
		{"write with session", "POST", "/api/v1/docker/start", []string{"Bearer machine-test-credential"}, false, 403},
		{"mutating GET", "GET", "/api/v1/cert/renew", []string{"Bearer machine-test-credential"}, false, 403},
		{"control upgrade", "GET", "/api/v1/cert/renew", []string{"Bearer machine-test-credential"}, true, 403},
		{"read upgrade", "GET", "/api/v1/route/list", []string{"Bearer machine-test-credential"}, true, 403},
		{"static sibling", "GET", "/api/v1/route/validate", []string{"Bearer machine-test-credential"}, false, 403},
		{"sensitive file read", "GET", "/api/v1/file/content", []string{"Bearer machine-test-credential"}, false, 403},
		{"session logout", "POST", "/api/v1/auth/logout", []string{"Bearer machine-test-credential"}, false, 403},
		{"future endpoint", "GET", "/api/v1/route/future", []string{"Bearer machine-test-credential"}, false, 403},
		{"unknown endpoint", "GET", "/api/v1/future", []string{"Bearer machine-test-credential"}, false, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil).WithContext(parent.Context())
			req.Header["Authorization"] = tc.authorization
			req.AddCookie(session)
			if tc.upgrade {
				req.Header.Set("Connection", "keep-alive, Upgrade")
				req.Header.Set("Upgrade", "websocket")
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tc.status, rec.Code)
			assert.Empty(t, rec.Result().Cookies())
			assert.NotContains(t, rec.Body.String(), "machine-test-credential")
		})
	}
	assert.False(t, called)

	// Cookie-only requests retain their broader session and CSRF policy.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/route/future", nil)
	req.AddCookie(session)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.True(t, called)
	assert.NotNil(t, findCookie(rec.Result().Cookies(), auth.CSRFCookieName))
	req = httptest.NewRequest(http.MethodPost, "/api/v1/docker/start", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing CSRF token")
}

func TestReadOnlyBearerConfigurationBoundaries(t *testing.T) {
	for _, name := range []string{"GODOXY_API_READONLY_TOKEN", "GOPROXY_API_READONLY_TOKEN", "API_READONLY_TOKEN"} {
		t.Setenv(name, "")
	}
	handler := newAuthenticatedHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/route/list", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.AddCookie(&http.Cookie{Name: "godoxy_token", Value: issueSessionToken(t)})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	t.Setenv("GODOXY_API_READONLY_TOKEN", "machine-test-credential")
	local := NewHandler(false)
	common.DebugDisableAuth = true // restored by newAuthenticatedHandler cleanup
	disabled := NewHandler(true)
	for name, h := range map[string]http.Handler{"local": local, "disabled": disabled} {
		t.Run(name, func(t *testing.T) {
			// This non-allowlisted route still works, even with an invalid header.
			req := httptest.NewRequest(http.MethodGet, "/api/v1/route/providers", nil)
			req.Header.Set("Authorization", "Bearer wrong")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Empty(t, rec.Result().Cookies())
		})
	}
}
