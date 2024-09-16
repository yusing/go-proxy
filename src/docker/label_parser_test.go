package docker

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	E "github.com/yusing/go-proxy/error"
)

func makeLabel(namespace string, alias string, field string) string {
	return fmt.Sprintf("%s.%s.%s", namespace, alias, field)
}

func TestHomePageLabel(t *testing.T) {
	alias := "foo"
	field := "ip"
	v := "bar"
	pl, err := ParseLabel(makeLabel(NSHomePage, alias, field), v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err.Error())
	}
	if pl.Target != alias {
		t.Errorf("expected alias=%s, got %s", alias, pl.Target)
	}
	if pl.Attribute != field {
		t.Errorf("expected field=%s, got %s", field, pl.Target)
	}
	if pl.Value != v {
		t.Errorf("expected value=%q, got %s", v, pl.Value)
	}
}

func TestStringProxyLabel(t *testing.T) {
	v := "bar"
	pl, err := ParseLabel(makeLabel(NSProxy, "foo", "ip"), v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err.Error())
	}
	if pl.Value != v {
		t.Errorf("expected value=%q, got %s", v, pl.Value)
	}
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
		if err.IsNotNil() {
			t.Errorf("expected err=nil, got %s", err.Error())
		}
		if pl.Value != v {
			t.Errorf("expected value=%v, got %v", v, pl.Value)
		}
	}
}

func TestBoolProxyLabelInvalid(t *testing.T) {
	alias := "foo"
	field := "no_tls_verify"
	_, err := ParseLabel(makeLabel(NSProxy, alias, field), "invalid")
	if !err.Is(E.ErrInvalid) {
		t.Errorf("expected err InvalidProxyLabel, got %s", err.Error())
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
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err.Error())
	}
	hGot, ok := pl.Value.(map[string]string)
	if !ok {
		t.Errorf("value is not a map[string]string, but %T", pl.Value)
		return
	}
	if !reflect.DeepEqual(h, hGot) {
		t.Errorf("expected %v, got %v", h, hGot)
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
			t.Errorf("expected invalid err for %q, got %s", v, err.Error())
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
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err.Error())
	}
	sGot, ok := pl.Value.([]string)
	sWant := []string{"X-Custom-Header1", "X-Custom-Header2", "X-Custom-Header3"}
	if !ok {
		t.Errorf("value is not []string, but %T", pl.Value)
	}
	if !reflect.DeepEqual(sGot, sWant) {
		t.Errorf("expected %q, got %q", sWant, sGot)
	}
}

func TestCommaSepProxyLabelSingle(t *testing.T) {
	v := "a"
	pl, err := ParseLabel("proxy.aliases", v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err.Error())
	}
	sGot, ok := pl.Value.([]string)
	sWant := []string{"a"}
	if !ok {
		t.Errorf("value is not []string, but %T", pl.Value)
	}
	if !reflect.DeepEqual(sGot, sWant) {
		t.Errorf("expected %q, got %q", sWant, sGot)
	}
}

func TestCommaSepProxyLabelMulti(t *testing.T) {
	v := "X-Custom-Header1, X-Custom-Header2,X-Custom-Header3"
	pl, err := ParseLabel("proxy.aliases", v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err.Error())
	}
	sGot, ok := pl.Value.([]string)
	sWant := []string{"X-Custom-Header1", "X-Custom-Header2", "X-Custom-Header3"}
	if !ok {
		t.Errorf("value is not []string, but %T", pl.Value)
	}
	if !reflect.DeepEqual(sGot, sWant) {
		t.Errorf("expected %q, got %q", sWant, sGot)
	}
}
