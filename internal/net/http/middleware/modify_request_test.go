package middleware

import (
	"slices"
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestSetModifyRequest(t *testing.T) {
	opts := OptionsRaw{
		"set_headers": map[string]string{
			"User-Agent": "go-proxy/v0.5.0",
			"Host":       "test.example.com",
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
		result, err := newMiddlewareTest(ModifyRequest, &testArgs{
			middlewareOpt: opts,
		})
		ExpectNoError(t, err)
		ExpectEqual(t, result.RequestHeaders.Get("User-Agent"), "go-proxy/v0.5.0")
		ExpectEqual(t, result.RequestHeaders.Get("Host"), "test.example.com")
		ExpectTrue(t, slices.Contains(result.RequestHeaders.Values("Accept-Encoding"), "test-value"))
		ExpectEqual(t, result.RequestHeaders.Get("Accept"), "")
	})
}
