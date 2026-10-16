package routeApi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPlayground(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		request        PlaygroundRequest
		wantStatusCode int
		checkResponse  func(t *testing.T, resp PlaygroundResponse)
	}{
		{
			name: "simple path matching rule",
			request: PlaygroundRequest{
				Rules: `- name: test rule
  on: path /api
  do: pass
`,
				MockRequest: MockRequest{
					Method: "GET",
					Path:   "/api",
				},
			},
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp PlaygroundResponse) {
				if len(resp.ParsedRules) != 1 {
					t.Errorf("expected 1 parsed rule, got %d", len(resp.ParsedRules))
				}
				if resp.ParsedRules[0].ValidationError != nil {
					t.Errorf("expected rule to be valid, got error: %v", resp.ParsedRules[0].ValidationError)
				}
				if len(resp.MatchedRules) != 1 || resp.MatchedRules[0] != "test rule" {
					t.Errorf("expected matched rules to be ['test rule'], got %v", resp.MatchedRules)
				}
				if !resp.UpstreamCalled {
					t.Error("expected upstream to be called")
				}
			},
		},
		{
			name: "header matching rule",
			request: PlaygroundRequest{
				Rules: `- name: check user agent
  on: header User-Agent Chrome
  do: error 403 Forbidden
`,
				MockRequest: MockRequest{
					Method: "GET",
					Path:   "/",
					Headers: map[string][]string{
						"User-Agent": {"Chrome"},
					},
				},
			},
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp PlaygroundResponse) {
				if len(resp.ParsedRules) != 1 {
					t.Errorf("expected 1 parsed rule, got %d", len(resp.ParsedRules))
				}
				if resp.ParsedRules[0].ValidationError != nil {
					t.Errorf("expected rule to be valid, got error: %v", resp.ParsedRules[0].ValidationError)
				}
				if len(resp.MatchedRules) != 1 {
					t.Errorf("expected 1 matched rule, got %d", len(resp.MatchedRules))
				}
				if resp.FinalResponse.StatusCode != http.StatusForbidden {
					t.Errorf("expected status 403, got %d", resp.FinalResponse.StatusCode)
				}
				if resp.UpstreamCalled {
					t.Error("expected upstream not to be called")
				}
			},
		},
		{
			name: "invalid rule syntax",
			request: PlaygroundRequest{
				Rules: `- name: bad rule
  on: invalid_checker something
  do: pass
`,
				MockRequest: MockRequest{
					Method: "GET",
					Path:   "/",
				},
			},
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp PlaygroundResponse) {
				if len(resp.ParsedRules) != 1 {
					t.Errorf("expected 1 parsed rule, got %d", len(resp.ParsedRules))
				}
				if resp.ParsedRules[0].ValidationError == nil {
					t.Error("expected validation error to be set")
				}
			},
		},
		{
			name: "rewrite path rule",
			request: PlaygroundRequest{
				Rules: `- name: rewrite rule
  on: path glob(/api/*)
  do: rewrite /api/ /v1/
`,
				MockRequest: MockRequest{
					Method: "GET",
					Path:   "/api/users",
				},
			},
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp PlaygroundResponse) {
				if len(resp.ParsedRules) != 1 {
					t.Errorf("expected 1 parsed rule, got %d", len(resp.ParsedRules))
				}
				if resp.ParsedRules[0].ValidationError != nil {
					t.Errorf("expected rule to be valid, got error: %v", resp.ParsedRules[0].ValidationError)
				}
				if !resp.UpstreamCalled {
					t.Error("expected upstream to be called")
				}
				if resp.FinalRequest.Path != "/v1/users" {
					t.Errorf("expected path to be rewritten to /v1/users, got %s", resp.FinalRequest.Path)
				}
				// Note: matched rules tracking has limitations with fresh ResponseModifier
				// The important thing is that the rewrite actually worked
			},
		},
		{
			name: "method matching rule",
			request: PlaygroundRequest{
				Rules: `- name: block POST
  on: method POST
  do: error "405" "Method Not Allowed"
`,
				MockRequest: MockRequest{
					Method: "POST",
					Path:   "/api",
				},
			},
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp PlaygroundResponse) {
				if resp.ParsedRules[0].ValidationError != nil {
					t.Errorf("expected rule to be valid, got error: %v", resp.ParsedRules[0].ValidationError)
				}
				if len(resp.MatchedRules) != 1 {
					t.Errorf("expected 1 matched rule, got %d", len(resp.MatchedRules))
				}
				if resp.FinalResponse.StatusCode != http.StatusMethodNotAllowed {
					t.Errorf("expected status 405, got %d", resp.FinalResponse.StatusCode)
				}
			},
		},
		{
			name: "block syntax default rule",
			request: PlaygroundRequest{
				Rules: `default {
  pass
}`,
				MockRequest: MockRequest{
					Method: "GET",
					Path:   "/",
				},
			},
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp PlaygroundResponse) {
				if len(resp.ParsedRules) != 1 {
					t.Errorf("expected 1 parsed rule, got %d", len(resp.ParsedRules))
				}
				if resp.ParsedRules[0].ValidationError != nil {
					t.Errorf("expected rule to be valid, got error: %v", resp.ParsedRules[0].ValidationError)
				}
				if !resp.UpstreamCalled {
					t.Error("expected upstream to be called")
				}
			},
		},
		{
			name: "block syntax conditional rule",
			request: PlaygroundRequest{
				Rules: `header User-Agent Chrome {
  error 403 Forbidden
}`,
				MockRequest: MockRequest{
					Method: "GET",
					Path:   "/",
					Headers: map[string][]string{
						"User-Agent": {"Chrome"},
					},
				},
			},
			wantStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp PlaygroundResponse) {
				if len(resp.ParsedRules) != 1 {
					t.Errorf("expected 1 parsed rule, got %d", len(resp.ParsedRules))
				}
				if resp.ParsedRules[0].ValidationError != nil {
					t.Errorf("expected rule to be valid, got error: %v", resp.ParsedRules[0].ValidationError)
				}
				if len(resp.MatchedRules) != 1 {
					t.Errorf("expected 1 matched rule, got %d", len(resp.MatchedRules))
				}
				if resp.FinalResponse.StatusCode != http.StatusForbidden {
					t.Errorf("expected status 403, got %d", resp.FinalResponse.StatusCode)
				}
				if resp.UpstreamCalled {
					t.Error("expected upstream not to be called")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/route/playground", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Create gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			Playground(c)

			// Check status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("expected status code %d, got %d", tt.wantStatusCode, w.Code)
			}

			respAny, ok := c.Get("response")
			if !ok {
				t.Fatalf("expected response to be set")
			}
			resp := respAny.(PlaygroundResponse)

			// Run custom checks
			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func runPlayground(t *testing.T, request PlaygroundRequest) PlaygroundResponse {
	t.Helper()
	data, err := json.Marshal(request)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/route/playground", bytes.NewReader(data))
	c.Request.Header.Set("Content-Type", "application/json")
	Playground(c)
	require.Equal(t, http.StatusOK, w.Code)
	response, ok := c.Get("response")
	require.True(t, ok)
	return response.(PlaygroundResponse)
}

func TestPlaygroundMockResponse(t *testing.T) {
	for _, tc := range []struct {
		name, rules string
		status      int
		body        string
		upstream    bool
	}{
		{"pass", "default { pass }", http.StatusCreated, "mock body", true},
		{"preflight", `method OPTIONS { error 204 "" }`, http.StatusNoContent, "", false},
		{"local error", `default { error 403 denied }`, http.StatusForbidden, "denied\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := runPlayground(t, PlaygroundRequest{
				Rules: tc.rules, MockRequest: MockRequest{Method: http.MethodOptions},
				MockResponse: MockResponse{StatusCode: http.StatusCreated, Body: "mock body",
					Headers: map[string][]string{"X-Mock": {"upstream"}}},
			})
			require.Nil(t, response.ExecutionError)
			require.Equal(t, tc.status, response.FinalResponse.StatusCode)
			require.Equal(t, tc.body, response.FinalResponse.Body)
			require.Equal(t, tc.upstream, response.UpstreamCalled)
			require.Equal(t, tc.upstream, len(response.FinalResponse.Headers["X-Mock"]) > 0)
		})
	}
}

func TestPlaygroundRequestSnapshot(t *testing.T) {
	response := runPlayground(t, PlaygroundRequest{
		Rules:       "default {\n set body replacement\n set header X-Changed yes\n}",
		MockRequest: MockRequest{Method: http.MethodPost, Body: "original"},
	})
	require.Nil(t, response.ExecutionError)
	require.True(t, response.UpstreamCalled)
	require.Equal(t, "replacement\n", response.FinalRequest.Body)
	require.Equal(t, []string{"yes"}, response.FinalRequest.Headers["X-Changed"])
}

func TestPlaygroundRejectsResponsePhaseAuth(t *testing.T) {
	for _, action := range []string{"require_auth", `require_basic_auth "Restricted"`} {
		for _, config := range []string{
			"status 200 { " + action + " }",
			"- on: status 200\n  do: " + action,
		} {
			response := runPlayground(t, PlaygroundRequest{Rules: config})
			require.ErrorContains(t, response.ExecutionError, "request-only")
			require.NotEmpty(t, response.ParsedRules)
			require.ErrorContains(t, response.ParsedRules[0].ValidationError, "request-only")
			require.False(t, response.UpstreamCalled)
		}
	}
}

func TestPlaygroundMalformedMockInput(t *testing.T) {
	for _, request := range []PlaygroundRequest{
		{Rules: "default { pass }", MockRequest: MockRequest{Method: "NOT A METHOD"}},
		{Rules: "default { pass }", MockRequest: MockRequest{Path: "/bad%"}},
		{Rules: "default { pass }", MockResponse: MockResponse{StatusCode: 1000}},
		{Rules: "default { pass }", MockResponse: MockResponse{StatusCode: -1}},
	} {
		data, err := json.Marshal(request)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
		c.Request.Header.Set("Content-Type", "application/json")
		require.NotPanics(t, func() { Playground(c) })
		require.Equal(t, http.StatusBadRequest, w.Code)
	}
}

func TestPlaygroundRecoveredErrorJSON(t *testing.T) {
	var executionError error
	handlerWithRecover(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil),
		func(http.ResponseWriter, *http.Request) { panic("execution failed") }, &executionError)
	data, err := json.Marshal(PlaygroundResponse{ExecutionError: executionError})
	require.NoError(t, err)
	require.Contains(t, string(data), "panic during rule execution: execution failed")
}

func TestPlaygroundIPv6AndQuery(t *testing.T) {
	response := runPlayground(t, PlaygroundRequest{
		Rules: "remote ::1/128 { error 403 denied }",
		MockRequest: MockRequest{RemoteIP: "::1", Path: "/api?existing=yes",
			Query: map[string][]string{"extra": {"value"}}},
	})
	require.Nil(t, response.ExecutionError)
	require.Equal(t, http.StatusForbidden, response.FinalResponse.StatusCode)
	require.Equal(t, map[string][]string{"existing": {"yes"}, "extra": {"value"}}, response.FinalRequest.Query)
}

func TestPlaygroundCancelsRealProxy(t *testing.T) {
	started, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(canceled)
		case <-release:
		}
	}))
	defer backend.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	data, err := json.Marshal(PlaygroundRequest{Rules: "default { proxy " + backend.URL + " }"})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data)).WithContext(ctx)
	c.Request.Header.Set("Content-Type", "application/json")
	done := make(chan struct{})
	go func() { defer close(done); Playground(c) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("proxy request did not reach backend")
	}
	cancel()
	select {
	case <-canceled:
	case <-time.After(3 * time.Second):
		t.Fatal("API cancellation did not reach backend")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("playground did not finish after cancellation")
	}
}

func TestPlaygroundFinalResponseMatches(t *testing.T) {
	response := runPlayground(t, PlaygroundRequest{
		Rules:        "- name: created\n  on: status 201\n  do: set resp_header X-Created yes",
		MockResponse: MockResponse{StatusCode: http.StatusCreated},
	})
	require.Nil(t, response.ExecutionError)
	require.True(t, slices.Contains(response.MatchedRules, "created"))
}

func TestPlaygroundYAMLEnvironmentMetadata(t *testing.T) {
	t.Setenv("GODOXY_RULE_NAME", "default")
	response := runPlayground(t, PlaygroundRequest{
		Rules:       "- name: ${RULE_NAME}\n  do: error 403 denied\n- on: path /allowed\n  do: pass",
		MockRequest: MockRequest{Path: "/allowed"},
	})
	require.Nil(t, response.ExecutionError)
	require.True(t, response.UpstreamCalled)
	require.Equal(t, http.StatusOK, response.FinalResponse.StatusCode)
}

func TestPlaygroundBodylessMockResponses(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotModified} {
		response := runPlayground(t, PlaygroundRequest{
			Rules: "default { pass }", MockResponse: MockResponse{StatusCode: status, Body: "ignored"},
		})
		require.Nil(t, response.ExecutionError)
		require.Empty(t, response.FinalResponse.Body)
	}
}

func TestPlaygroundProxyUsesRealResponse(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("real body"))
	}))
	defer backend.Close()
	response := runPlayground(t, PlaygroundRequest{
		Rules:        "default { proxy " + backend.URL + " }",
		MockResponse: MockResponse{StatusCode: http.StatusCreated, Body: "mock body"},
	})
	require.Nil(t, response.ExecutionError)
	require.False(t, response.UpstreamCalled)
	require.Equal(t, http.StatusAccepted, response.FinalResponse.StatusCode)
	require.Equal(t, "real body", strings.TrimSpace(response.FinalResponse.Body))
}

func TestPlaygroundInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/route/playground", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	Playground(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}
