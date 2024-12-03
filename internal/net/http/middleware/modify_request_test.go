package middleware

import (
	"bytes"
	"net/http"
	"slices"
	"testing"

	"github.com/yusing/go-proxy/internal/net/types"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestSetModifyRequest(t *testing.T) {
	opts := OptionsRaw{
		"set_headers": map[string]string{
			"User-Agent":                "go-proxy/v0.5.0",
			"Host":                      "$upstream_addr",
			"X-Test-Req-Method":         "$req_method",
			"X-Test-Req-Scheme":         "$req_scheme",
			"X-Test-Req-Host":           "$req_host",
			"X-Test-Req-Port":           "$req_port",
			"X-Test-Req-Addr":           "$req_addr",
			"X-Test-Req-Path":           "$req_path",
			"X-Test-Req-Query":          "$req_query",
			"X-Test-Req-Url":            "$req_url",
			"X-Test-Req-Uri":            "$req_uri",
			"X-Test-Req-Content-Type":   "$req_content_type",
			"X-Test-Req-Content-Length": "$req_content_length",
			"X-Test-Remote-Addr":        "$remote_addr",
			"X-Test-Upstream-Scheme":    "$upstream_scheme",
			"X-Test-Upstream-Host":      "$upstream_host",
			"X-Test-Upstream-Port":      "$upstream_port",
			"X-Test-Upstream-Addr":      "$upstream_addr",
			"X-Test-Upstream-Url":       "$upstream_url",
			"X-Test-Content-Type":       "$header(Content-Type)",
			"X-Test-Arg-Arg_1":          "$arg(arg_1)",
		},
		"add_headers":  map[string]string{"Accept-Encoding": "test-value"},
		"hide_headers": []string{"Accept"},
	}

	t.Run("set_options", func(t *testing.T) {
		mr, err := ModifyRequest.WithOptionsClone(opts)
		ExpectNoError(t, err)
		ExpectDeepEqual(t, mr.impl.(*modifyRequest).SetHeaders, opts["set_headers"].(map[string]string))
		ExpectDeepEqual(t, mr.impl.(*modifyRequest).AddHeaders, opts["add_headers"].(map[string]string))
		ExpectDeepEqual(t, mr.impl.(*modifyRequest).HideHeaders, opts["hide_headers"].([]string))
	})

	t.Run("request_headers", func(t *testing.T) {
		reqURL := types.MustParseURL("https://my.app/?arg_1=b")
		upstreamURL := types.MustParseURL("http://test.example.com")
		result, err := newMiddlewareTest(ModifyRequest, &testArgs{
			middlewareOpt: opts,
			reqURL:        reqURL,
			upstreamURL:   upstreamURL,
			body:          bytes.Repeat([]byte("a"), 100),
			headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
		})
		ExpectNoError(t, err)
		ExpectEqual(t, result.RequestHeaders.Get("User-Agent"), "go-proxy/v0.5.0")
		ExpectEqual(t, result.RequestHeaders.Get("Host"), "test.example.com")
		ExpectTrue(t, slices.Contains(result.RequestHeaders.Values("Accept-Encoding"), "test-value"))
		ExpectEqual(t, result.RequestHeaders.Get("Accept"), "")

		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Method"), "GET")
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Scheme"), reqURL.Scheme)
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Host"), reqURL.Hostname())
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Port"), reqURL.Port())
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Addr"), reqURL.Host)
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Path"), reqURL.Path)
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Query"), reqURL.RawQuery)
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Url"), reqURL.String())
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Uri"), reqURL.RequestURI())
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Content-Type"), "application/json")
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Req-Content-Length"), "100")
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Remote-Addr"), result.RemoteAddr)

		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Upstream-Scheme"), upstreamURL.Scheme)
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Upstream-Host"), upstreamURL.Hostname())
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Upstream-Port"), upstreamURL.Port())
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Upstream-Addr"), upstreamURL.Host)
		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Upstream-Url"), upstreamURL.String())

		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Content-Type"), "application/json")

		ExpectEqual(t, result.RequestHeaders.Get("X-Test-Arg-Arg_1"), "b")
	})
}
