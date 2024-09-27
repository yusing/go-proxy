package middleware

import (
	"slices"
	"testing"

	. "github.com/yusing/go-proxy/utils/testing"
)

func TestSetModifyResponse(t *testing.T) {
	opts := OptionsRaw{
		"set_headers":  map[string]string{"User-Agent": "go-proxy/v0.5.0"},
		"add_headers":  map[string]string{"Accept-Encoding": "test-value"},
		"hide_headers": []string{"Accept"},
	}

	t.Run("set_options", func(t *testing.T) {
		mr, err := ModifyResponse.m.WithOptionsClone(opts, nil)
		ExpectNoError(t, err.Error())
		ExpectDeepEqual(t, mr.impl.(*modifyResponse).SetHeaders, opts["set_headers"].(map[string]string))
		ExpectDeepEqual(t, mr.impl.(*modifyResponse).AddHeaders, opts["add_headers"].(map[string]string))
		ExpectDeepEqual(t, mr.impl.(*modifyResponse).HideHeaders, opts["hide_headers"].([]string))
	})

	t.Run("request_headers", func(t *testing.T) {
		result, err := newMiddlewareTest(ModifyResponse.m, &testArgs{
			middlewareOpt: opts,
		})
		ExpectNoError(t, err.Error())
		ExpectEqual(t, result.ResponseHeaders.Get("User-Agent"), "go-proxy/v0.5.0")
		t.Log(result.ResponseHeaders.Get("Accept-Encoding"))
		ExpectTrue(t, slices.Contains(result.ResponseHeaders.Values("Accept-Encoding"), "test-value"))
		ExpectEqual(t, result.ResponseHeaders.Get("Accept"), "")
	})
}
