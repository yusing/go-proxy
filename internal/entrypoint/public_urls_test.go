package entrypoint_test

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	homepageapi "github.com/yusing/godoxy/internal/api/v1/homepage"
	routeapi "github.com/yusing/godoxy/internal/api/v1/route"
	"github.com/yusing/godoxy/internal/common"
	"github.com/yusing/godoxy/internal/entrypoint"
	"github.com/yusing/godoxy/internal/homepage"
	"github.com/yusing/godoxy/internal/route"
	"github.com/yusing/godoxy/internal/routeimpl"
	"github.com/yusing/godoxy/internal/routetest"
	"github.com/yusing/goutils/task"
)

func TestPublicURLConsumers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldHTTP, oldHTTPS := common.ProxyHTTPAddr, common.ProxyHTTPSAddr
	t.Cleanup(func() { common.ProxyHTTPAddr, common.ProxyHTTPSAddr = oldHTTP, oldHTTPS })
	for _, tc := range []struct {
		name, alias, httpAddr, httpsAddr, override string
		domains                                    []string
		port                                       int
		scheme                                     route.Scheme
		excluded                                   bool
		want                                       []string
		homepageURL                                string
	}{
		{name: "multi-domain dotted short", alias: "app.dev", domains: []string{"example.com", ".example.net"}, want: []string{"https://app.dev.example.com", "https://app.dev.example.net"}},
		{name: "short", alias: "app", domains: []string{"example.com"}, want: []string{"https://app.example.com"}},
		{name: "fqdn", alias: "app.example.com", domains: []string{"example.net", "example.com"}, want: []string{"https://app.example.com"}},
		{name: "apex", alias: "example.com", domains: []string{"example.com"}, want: []string{"https://example.com"}},
		{name: "label boundary", alias: "notexample.com", domains: []string{"example.com"}, want: []string{"https://notexample.com.example.com"}},
		{name: "no domains fqdn", alias: "app.dev", want: []string{"https://app.dev"}},
		{name: "no domains short", alias: "app", homepageURL: "http://app.caller.test:9000"},
		{name: "https port", alias: "app", domains: []string{"example.com"}, httpsAddr: ":8443", want: []string{"https://app.example.com:8443"}},
		{name: "http default", alias: "app", domains: []string{"example.com"}, httpAddr: ":80", want: []string{"http://app.example.com"}},
		{name: "http port", alias: "app", domains: []string{"example.com"}, httpAddr: ":8080", want: []string{"http://app.example.com:8080"}},
		{name: "dedicated https listener", alias: "app", domains: []string{"example.com"}, httpAddr: ":80", port: 9443, want: []string{"https://app.example.com:9443"}},
		{name: "homepage override", alias: "app.dev", domains: []string{"example.com"}, override: "https://custom.test/path", want: []string{"https://app.dev.example.com"}, homepageURL: "https://custom.test/path"},
		{name: "excluded", alias: "app", domains: []string{"example.com"}, scheme: route.SchemeHTTP, excluded: true},
		{name: "tcp", alias: "app", domains: []string{"example.com"}, scheme: route.SchemeTCP},
		{name: "udp", alias: "app", domains: []string{"example.com"}, scheme: route.SchemeUDP},
		{name: "fileserver", alias: "app", domains: []string{"example.com"}, scheme: route.SchemeFileServer, want: []string{"https://app.example.com"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			common.ProxyHTTPAddr, common.ProxyHTTPSAddr = ":80", ":443"
			if tc.httpAddr != "" {
				common.ProxyHTTPAddr, common.ProxyHTTPSAddr = tc.httpAddr, ""
			}
			if tc.httpsAddr != "" {
				common.ProxyHTTPSAddr = tc.httpsAddr
			}
			ep := entrypoint.NewTestEntrypoint(t, nil)
			ep.SetFindRouteDomains(tc.domains)
			base := &route.Route{
				Alias: tc.alias, Scheme: tc.scheme, Port: route.Port{Listening: tc.port},
				Homepage: &homepage.ItemConfig{Name: tc.alias, Show: true, URL: tc.override},
				Metadata: route.Metadata{Provider: "fixture"},
			}
			if tc.excluded {
				base.Excluded = true
				base.ExcludedReason = route.ExcludedReasonManual
				ep.ExcludedRoutes().Add(&routeimpl.ReverseProxyRoute{Route: base})
			} else if tc.scheme == route.SchemeTCP || tc.scheme == route.SchemeUDP {
				entrypoint.AddTestStreamRoute(ep, &routeimpl.StreamRoute{Route: base})
			} else {
				_, err := routetest.NewStartedRoute(t, base)
				require.NoError(t, err)
			}
			router := gin.New()
			router.GET("/route/list", routeapi.Routes)
			router.GET("/route/:which", routeapi.Route)
			router.GET("/homepage/items", homepageapi.Items)
			get := func(path, host string) []byte {
				t.Helper()
				req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(task.GetTestTask(t).Context())
				req.Host = host
				req.Header.Set("X-Forwarded-Host", host)
				req.Header.Set("X-Forwarded-Proto", "http")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
				return w.Body.Bytes()
			}
			for _, host := range []string{"127.0.0.1:8888", "unrelated.test"} {
				for _, path := range []string{"/route/list", "/route/list?provider=fixture", "/route/" + base.Key()} {
					data := get(path, host)
					var payload []struct {
						Alias string   `json:"alias"`
						URLs  []string `json:"public_urls"`
					}
					if data[0] != '[' {
						data = append(append([]byte{'['}, data...), ']')
					}
					require.NoError(t, json.Unmarshal(data, &payload))
					require.Len(t, payload, 1)
					require.Equal(t, tc.alias, payload[0].Alias)
					require.Equal(t, tc.want, payload[0].URLs, path)
					var fields []map[string]jsontext.Value
					require.NoError(t, json.Unmarshal(data, &fields))
					for _, field := range []string{"no_tls_verify", "response_header_timeout", "max_conns_per_host", "disable_compression", "spa"} {
						require.NotContains(t, fields[0], field, "HTTP retains JSON v1 zero-value omission")
					}
				}
			}
			var categories homepage.Homepage
			require.NoError(t, json.Unmarshal(get("/homepage/items", "dashboard.caller.test:9000"), &categories))
			items := make(map[string]string)
			for _, category := range categories {
				for _, item := range category.Items {
					items[item.Alias] = item.URL
				}
			}
			if tc.excluded || tc.scheme == route.SchemeTCP || tc.scheme == route.SchemeUDP {
				require.Empty(t, items)
			} else {
				want := tc.homepageURL
				if want == "" {
					want = tc.want[0]
				}
				require.Equal(t, map[string]string{tc.alias: want}, items)
			}
		})
	}
}

func TestRouteWebsocketPublicURLUpdates(t *testing.T) {
	ep := entrypoint.NewTestEntrypoint(t, nil)
	ep.SetFindRouteDomains([]string{"example.com"})
	addRoute(t, "first")
	router := gin.New()
	router.GET("/route/list", routeapi.Routes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(task.GetTestTask(t).Context()))
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/route/list", nil)
	require.NoError(t, err)
	defer conn.Close()
	read := func(alias string) {
		t.Helper()
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
		_, data, err := conn.ReadMessage()
		require.NoError(t, err)
		var payload []map[string]jsontext.Value
		require.NoError(t, json.Unmarshal(data, &payload))
		require.Len(t, payload, 1)
		var urls []string
		require.NoError(t, json.Unmarshal(payload[0]["public_urls"], &urls))
		require.Equal(t, []string{"https://" + alias + ".example.com:18080"}, urls)
		for field, want := range map[string]string{
			"no_tls_verify": "false", "response_header_timeout": "0", "max_conns_per_host": "0",
			"disable_compression": "false", "spa": "false",
		} {
			require.Equal(t, want, string(payload[0][field]), "WebSocket retains JSON v2 field %s", field)
		}
	}
	read("first")
	old, ok := ep.HTTPRoutes().Get("first")
	require.True(t, ok)
	listener, ok := ep.GetServer(":18080")
	require.True(t, ok)
	listener.DelRoute(old)
	addRoute(t, "second")
	read("second")
}
