package provider

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yusing/godoxy/internal/common"
	"github.com/yusing/godoxy/internal/entrypoint"
	"github.com/yusing/godoxy/internal/health"
	idlewatcher "github.com/yusing/godoxy/internal/idlewatcher/runtime"
	"github.com/yusing/godoxy/internal/proxmox"
	"github.com/yusing/godoxy/internal/route"
)

func TestProxmoxStartupRetryRegistersIdlewatcherRoute(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("application response"))
	}))
	t.Cleanup(upstream.Close)
	var ticketRequests, controlRequests atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch strings.TrimPrefix(r.URL.Path, "/api2/json") {
		case "/access/ticket":
			if ticketRequests.Add(1) == 1 {
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				_ = conn.Close()
				return
			}
			_, _ = w.Write([]byte(`{"data":{"username":"godoxy@pam","ticket":"TEST-TICKET","CSRFPreventionToken":"TEST-CSRF"}}`))
		case "/version":
			_, _ = w.Write([]byte(`{"data":{"release":"9.1","version":"9.1-1"}}`))
		case "/cluster/status":
			_, _ = w.Write([]byte(`{"data":[{"type":"node","name":"pve","id":"node/pve","nodeid":1,"online":1}]}`))
		case "/cluster/resources":
			_, _ = w.Write([]byte(`{"data":[{"id":"lxc/101","name":"app","node":"pve","status":"running","vmid":101}]}`))
		case "/nodes/pve/lxc/101/interfaces":
			_, _ = w.Write([]byte(`{"data":[{"name":"eth0","inet":"10.0.0.101/24"}]}`))
		case "/nodes/pve/lxc/101/status/current":
			_, _ = w.Write([]byte(`{"data":{"name":"app","status":"running"}}`))
		case "/nodes/pve/lxc/101/config":
			_, _ = w.Write([]byte(`{"data":null}`))
		default:
			if strings.Contains(r.URL.Path, "/status/") && r.Method == http.MethodPost {
				controlRequests.Add(1)
			}
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)
	parent := newRecoveryRuntime(t)
	proxmox.SetCtx(parent, proxmox.NewNodePool())
	configured := &proxmox.Config{URL: api.URL, Username: "godoxy", Password: "test-password"}

	require.NoError(t, configured.Init(parent.Context()))
	require.EqualValues(t, 2, ticketRequests.Load())
	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(target.Port())
	require.NoError(t, err)
	vmid := uint64(101)
	p := NewStaticProvider("startup-test", route.Routes{"app": {
		Scheme:      route.SchemeHTTP,
		Host:        target.Hostname(),
		Port:        route.Port{Proxy: port},
		Proxmox:     &proxmox.NodeConfig{Node: "pve", VMID: &vmid},
		Idlewatcher: &idlewatcher.Config{IdlewatcherConfigBase: idlewatcher.ConfigBase{IdleTimeout: 32 * time.Minute}},
	}})
	require.NoError(t, p.LoadRoutes(parent.Context()))
	activation := p.Activate(parent)
	require.Equal(t, 1, activation.ActiveRoutes)
	require.Empty(t, activation.FailedRoutes)
	ep := entrypoint.FromCtx(parent.Context())
	active, ok := ep.HTTPRoutes().Get("app")
	require.True(t, ok, "the route must appear in the active route list after the transient startup failure")
	require.Eventually(t, func() bool {
		return active.HealthMonitor().Status() == health.StatusHealthy
	}, 3*time.Second, 10*time.Millisecond)
	server, ok := ep.(*entrypoint.Entrypoint).GetServer(common.ProxyHTTPAddr)
	require.True(t, ok)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://app/", nil))
	require.Equal(t, http.StatusOK, response.Code, "the recovered hostname must no longer return 404")
	require.Equal(t, "application response", response.Body.String())
	require.Zero(t, controlRequests.Load(), "initialization must not start or stop an already running LXC")
	parent.FinishAndWait(nil)
}
