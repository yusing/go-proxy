package proxmox

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	goproxmox "github.com/luthermonson/go-proxmox"
	"github.com/stretchr/testify/require"
)

func writeStartupResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch strings.TrimPrefix(r.URL.Path, "/api2/json") {
	case "/access/ticket":
		_, _ = w.Write([]byte(`{"data":{"username":"godoxy@pam","ticket":"TEST-TICKET","CSRFPreventionToken":"TEST-CSRF"}}`))
	case "/version":
		_, _ = w.Write([]byte(`{"data":{"repoid":"test","release":"9.1","version":"9.1-1"}}`))
	case "/cluster/status":
		_, _ = w.Write([]byte(`{"data":[{"type":"node","name":"pve","id":"node/pve","nodeid":1,"online":1,"ip":"192.0.2.1"}]}`))
	case "/cluster/resources":
		_, _ = w.Write([]byte(`{"data":[]}`))
	default:
		http.NotFound(w, r)
	}
}

func TestConfigInitRetriesTransientStartupFailures(t *testing.T) {
	for _, tc := range []struct {
		name        string
		path        string
		credentials bool
	}{
		{"session", "/access/ticket", true},
		{"version after session", "/version", true},
		{"cluster after session", "/cluster/status", true},
		{"API token", "/version", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var attempts, tickets atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimPrefix(r.URL.Path, "/api2/json")
				if path == "/access/ticket" {
					tickets.Add(1)
				}
				// Two disconnects also defeat net/http's automatic GET retry on
				// a previously used connection, exercising GoDoxy's startup retry.
				if path == tc.path && attempts.Add(1) <= 2 {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					_ = conn.Close()
					return
				}
				writeStartupResponse(w, r)
			}))
			t.Cleanup(srv.Close)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			t.Cleanup(cancel)
			ctx = context.WithValue(ctx, nodePoolContextKey{}, NewNodePool())
			cfg := &Config{URL: srv.URL, TokenID: "test@pam!godoxy", Secret: "test-secret"}
			if tc.credentials {
				cfg.Username, cfg.Password = "godoxy", "test-password"
			}

			require.NoError(t, cfg.Init(ctx))
			node, err := NodeFromCtx(ctx, "pve")
			require.NoError(t, err, "idlewatcher routes need the initialized node in their runtime context")
			require.Same(t, cfg.Client(), node.Client())
			require.GreaterOrEqual(t, attempts.Load(), int32(3))
			if tc.credentials && tc.path != "/access/ticket" {
				require.EqualValues(t, 1, tickets.Load(), "a metadata retry must reuse the successful session")
			}
			if !tc.credentials {
				require.Zero(t, tickets.Load(), "token authentication must not create a password session")
			}
		})
	}
}

func TestConfigInitStartupRetryHonorsDeadline(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	ctx = context.WithValue(ctx, nodePoolContextKey{}, NewNodePool())
	cfg := &Config{URL: srv.URL, Username: "godoxy", Password: "test-password"}

	require.ErrorIs(t, cfg.Init(ctx), context.DeadlineExceeded)
	require.Greater(t, attempts.Load(), int32(0))
	_, err := NodeFromCtx(ctx, "pve")
	require.ErrorIs(t, err, ErrNodeNotFound, "failed initialization must not publish a ready node")
}

func TestConfigInitStartupRetryHonorsCancellation(t *testing.T) {
	var attempts atomic.Int32
	disconnected := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		first := attempts.Add(1) == 1
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
		if first {
			close(disconnected)
		}
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cfg := &Config{URL: srv.URL, Username: "godoxy", Password: "test-password"}
	done := make(chan error, 1)
	go func() { done <- cfg.Init(ctx) }()
	select {
	case <-disconnected:
		cancel()
	case <-time.After(time.Second):
		t.Fatal("initial connection was not attempted")
	}
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("startup retry did not stop after cancellation")
	}
	require.EqualValues(t, 1, attempts.Load(), "cancellation must prevent another connection attempt")
}

func TestConfigInitDoesNotRetryPermanentStartupErrors(t *testing.T) {
	t.Run("credentials rejected", func(t *testing.T) {
		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(http.StatusUnauthorized)
		}))
		t.Cleanup(srv.Close)
		cfg := &Config{URL: srv.URL, Username: "godoxy", Password: "test-password"}

		require.ErrorIs(t, cfg.Init(t.Context()), goproxmox.ErrNotAuthorized)
		require.EqualValues(t, 1, attempts.Load())
	})
	t.Run("invalid API response", func(t *testing.T) {
		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			_, _ = w.Write([]byte("not a Proxmox API response"))
		}))
		t.Cleanup(srv.Close)
		cfg := &Config{URL: srv.URL, TokenID: "test@pam!godoxy", Secret: "test-secret"}

		require.Error(t, cfg.Init(t.Context()))
		require.EqualValues(t, 1, attempts.Load())
	})
	t.Run("untrusted certificate", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(writeStartupResponse))
		t.Cleanup(srv.Close)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		cfg := &Config{URL: srv.URL, Username: "godoxy", Password: "test-password"}

		var certificateError x509.UnknownAuthorityError
		err := cfg.Init(ctx)
		require.True(t, errors.As(err, &certificateError), "certificate failures must remain visible immediately")
		require.NotErrorIs(t, err, context.DeadlineExceeded)
	})
}
