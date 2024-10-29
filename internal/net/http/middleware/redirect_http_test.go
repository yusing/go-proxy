package middleware

import (
	"net/http"
	"testing"

	"github.com/yusing/go-proxy/internal/common"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestRedirectToHTTPs(t *testing.T) {
	result, err := newMiddlewareTest(RedirectHTTP, &testArgs{
		scheme: "http",
	})
	ExpectNoError(t, err)
	ExpectEqual(t, result.ResponseStatus, http.StatusTemporaryRedirect)
	ExpectEqual(t, result.ResponseHeaders.Get("Location"), "https://"+testHost+":"+common.ProxyHTTPSPort)
}

func TestNoRedirect(t *testing.T) {
	result, err := newMiddlewareTest(RedirectHTTP, &testArgs{
		scheme: "https",
	})
	ExpectNoError(t, err)
	ExpectEqual(t, result.ResponseStatus, http.StatusOK)
}
