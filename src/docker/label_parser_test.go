package docker

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	E "github.com/yusing/go-proxy/error"
	. "github.com/yusing/go-proxy/utils/testing"
)

func makeLabel(namespace string, alias string, field string) string {
	return fmt.Sprintf("%s.%s.%s", namespace, alias, field)
}

func TestHomePageLabel(t *testing.T) {
	alias := "foo"
	field := "ip"
	v := "bar"
	pl, err := ParseLabel(makeLabel(NSHomePage, alias, field), v)
	ExpectNoError(t, err.Error())
	if pl.Target != alias {
		t.Errorf("Expected alias=%s, got %s", alias, pl.Target)
	}
	if pl.Attribute != field {
		t.Errorf("Expected field=%s, got %s", field, pl.Target)
	}
	if pl.Value != v {
		t.Errorf("Expected value=%q, got %s", v, pl.Value)
	}
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
		pl, err := ParseLabel(makeLabel(NSProxy, "foo", "no_tls_verify"), k)
		ExpectNoError(t, err.Error())
		ExpectEqual(t, pl.Value.(bool), v)
	}
}

func TestBoolProxyLabelInvalid(t *testing.T) {
	alias := "foo"
	field := "no_tls_verify"
	_, err := ParseLabel(makeLabel(NSProxy, alias, field), "invalid")
	if !err.Is(E.ErrInvalid) {
		t.Errorf("Expected err InvalidProxyLabel, got %s", err.Error())
	}
}

func TestSetHeaderProxyLabelValid(t *testing.T) {
	v := `
X-Custom-Header1: foo, bar
X-Custom-Header1: baz
X-Custom-Header2: boo`
	v = strings.TrimPrefix(v, "\n")
	h := map[string]string{
		"X-Custom-Header1": "foo, bar, baz",
		"X-Custom-Header2": "boo",
	}

	pl, err := ParseLabel(makeLabel(NSProxy, "foo", "set_headers"), v)
	ExpectNoError(t, err.Error())
	hGot := ExpectType[map[string]string](t, pl.Value)
	if hGot != nil && !reflect.DeepEqual(h, hGot) {
		t.Errorf("Expected %v, got %v", h, hGot)
	}

}

func TestSetHeaderProxyLabelInvalid(t *testing.T) {
	tests := []string{
		"X-Custom-Header1 = bar",
		"X-Custom-Header1",
		"- X-Custom-Header1",
	}

	for _, v := range tests {
		_, err := ParseLabel(makeLabel(NSProxy, "foo", "set_headers"), v)
		if !err.Is(E.ErrInvalid) {
			t.Errorf("Expected invalid err for %q, got %s", v, err.Error())
		}
	}
}

func TestHideHeadersProxyLabel(t *testing.T) {
	v := `
- X-Custom-Header1
- X-Custom-Header2
- X-Custom-Header3
`
	v = strings.TrimPrefix(v, "\n")
	pl, err := ParseLabel(makeLabel(NSProxy, "foo", "hide_headers"), v)
	ExpectNoError(t, err.Error())
	sGot := ExpectType[[]string](t, pl.Value)
	sWant := []string{"X-Custom-Header1", "X-Custom-Header2", "X-Custom-Header3"}
	if sGot != nil {
		ExpectDeepEqual(t, sGot, sWant)
	}
}

// func TestCommaSepProxyLabelSingle(t *testing.T) {
// 	v := "a"
// 	pl, err := ParseLabel("proxy.aliases", v)
// 	ExpectNoError(t, err)
// 	sGot := ExpectType[[]string](t, pl.Value)
// 	sWant := []string{"a"}
// 	if sGot != nil {
// 		ExpectEqual(t, sGot, sWant)
// 	}
// }

// func TestCommaSepProxyLabelMulti(t *testing.T) {
// 	v := "X-Custom-Header1, X-Custom-Header2,X-Custom-Header3"
// 	pl, err := ParseLabel("proxy.aliases", v)
// 	ExpectNoError(t, err)
// 	sGot := ExpectType[[]string](t, pl.Value)
// 	sWant := []string{"X-Custom-Header1", "X-Custom-Header2", "X-Custom-Header3"}
// 	if sGot != nil {
// 		ExpectEqual(t, sGot, sWant)
// 	}
// }
