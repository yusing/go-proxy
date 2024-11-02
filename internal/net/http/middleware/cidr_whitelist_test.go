package middleware

import (
	_ "embed"
	"net/http"
	"testing"

	E "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

//go:embed test_data/cidr_whitelist_test.yml
var testCIDRWhitelistCompose []byte
var deny, accept *Middleware

func TestCIDRWhitelist(t *testing.T) {
	errs := E.NewBuilder("")
	mids := BuildMiddlewaresFromYAML("", testCIDRWhitelistCompose, errs)
	ExpectNoError(t, errs.Error())
	deny = mids["deny@file"]
	accept = mids["accept@file"]
	if deny == nil || accept == nil {
		panic("bug occurred")
	}

	t.Run("deny", func(t *testing.T) {
		for range 10 {
			result, err := newMiddlewareTest(deny, nil)
			ExpectNoError(t, err)
			ExpectEqual(t, result.ResponseStatus, cidrWhitelistDefaults.StatusCode)
			ExpectEqual(t, string(result.Data), cidrWhitelistDefaults.Message)
		}
	})

	t.Run("accept", func(t *testing.T) {
		for range 10 {
			result, err := newMiddlewareTest(accept, nil)
			ExpectNoError(t, err)
			ExpectEqual(t, result.ResponseStatus, http.StatusOK)
		}
	})
}
