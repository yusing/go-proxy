package docker

import (
	"fmt"
	"testing"

	E "github.com/yusing/go-proxy/error"
	. "github.com/yusing/go-proxy/utils/testing"
)

func makeLabel(namespace string, alias string, field string) string {
	return fmt.Sprintf("%s.%s.%s", namespace, alias, field)
}

func TestParseLabel(t *testing.T) {
	alias := "foo"
	field := "ip"
	v := "bar"
	pl, err := ParseLabel(makeLabel(NSHomePage, alias, field), v)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, pl.Namespace, NSHomePage)
	ExpectEqual(t, pl.Target, alias)
	ExpectEqual(t, pl.Attribute, field)
	ExpectEqual(t, pl.Value.(string), v)
}

func TestStringProxyLabel(t *testing.T) {
	v := "bar"
	pl, err := ParseLabel(makeLabel(NSProxy, "foo", "ip"), v)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, pl.Value.(string), v)
}

func TestBoolProxyLabelValid(t *testing.T) {
	tests := map[string]bool{
		"true":  true,
		"TRUE":  true,
		"yes":   true,
		"1":     true,
		"false": false,
		"FALSE": false,
		"no":    false,
		"0":     false,
	}

	for k, v := range tests {
		pl, err := ParseLabel(makeLabel(NSProxy, "foo", ProxyAttributeNoTLSVerify), k)
		ExpectNoError(t, err.Error())
		ExpectEqual(t, pl.Value.(bool), v)
	}
}

func TestBoolProxyLabelInvalid(t *testing.T) {
	_, err := ParseLabel(makeLabel(NSProxy, "foo", ProxyAttributeNoTLSVerify), "invalid")
	if !err.Is(E.ErrInvalid) {
		t.Errorf("Expected err InvalidProxyLabel, got %s", err.Error())
	}
}

// func TestSetHeaderProxyLabelValid(t *testing.T) {
// 	v := `
// X-Custom-Header1: foo, bar
// X-Custom-Header1: baz
// X-Custom-Header2: boo`
// 	v = strings.TrimPrefix(v, "\n")
// 	h := map[string]string{
// 		"X-Custom-Header1": "foo, bar, baz",
// 		"X-Custom-Header2": "boo",
// 	}

// 	pl, err := ParseLabel(makeLabel(NSProxy, "foo", ProxyAttributeSetHeaders), v)
// 	ExpectNoError(t, err.Error())
// 	hGot := ExpectType[map[string]string](t, pl.Value)
// 	ExpectFalse(t, hGot == nil)
// 	ExpectDeepEqual(t, h, hGot)
// }

// func TestSetHeaderProxyLabelInvalid(t *testing.T) {
// 	tests := []string{
// 		"X-Custom-Header1 = bar",
// 		"X-Custom-Header1",
// 		"- X-Custom-Header1",
// 	}

// 	for _, v := range tests {
// 		_, err := ParseLabel(makeLabel(NSProxy, "foo", ProxyAttributeSetHeaders), v)
// 		if !err.Is(E.ErrInvalid) {
// 			t.Errorf("Expected invalid err for %q, got %s", v, err.Error())
// 		}
// 	}
// }

// func TestHideHeadersProxyLabel(t *testing.T) {
// 	v := `
// - X-Custom-Header1
// - X-Custom-Header2
// - X-Custom-Header3
// `
// 	v = strings.TrimPrefix(v, "\n")
// 	pl, err := ParseLabel(makeLabel(NSProxy, "foo", ProxyAttributeHideHeaders), v)
// 	ExpectNoError(t, err.Error())
// 	sGot := ExpectType[[]string](t, pl.Value)
// 	sWant := []string{"X-Custom-Header1", "X-Custom-Header2", "X-Custom-Header3"}
// 	ExpectFalse(t, sGot == nil)
// 	ExpectDeepEqual(t, sGot, sWant)
// }
