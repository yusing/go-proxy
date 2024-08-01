package docker

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	E "github.com/yusing/go-proxy/error"
)

func makeLabel(namespace string, alias string, field string) string {
	return fmt.Sprintf("%s.%s.%s", namespace, alias, field)
}

func TestInvalidLabel(t *testing.T) {
	pl, err := ParseLabel("foo.bar", "1234")
	if !errors.Is(err, ErrInvalidLabel) {
		t.Errorf("expected errInvalidLabel, got %s", err)
	}
	if pl != nil {
		t.Errorf("expected nil, got %v", pl)
	}
	_, err = ParseLabel("proxy.foo", "bar")
	if !errors.Is(err, ErrInvalidLabel) {
		t.Errorf("expected errInvalidLabel, got %s", err)
	}
}

func TestHomePageLabel(t *testing.T) {
	alias := "foo"
	field := "ip"
	v := "bar"
	pl, err := ParseLabel(makeLabel(NSHomePage, alias, field), v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err)
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
	alias := "foo"
	field := "ip"
	v := "bar"
	pl, err := ParseLabel(makeLabel(NSProxy, alias, field), v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err)
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

func TestBoolProxyLabelValid(t *testing.T) {
	alias := "foo"
	field := "no_tls_verify"
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
		pl, err := ParseLabel(makeLabel(NSProxy, alias, field), k)
		if err.IsNotNil() {
			t.Errorf("expected err=nil, got %s", err)
		}
		if pl.Target != alias {
			t.Errorf("expected alias=%s, got %s", alias, pl.Target)
		}
		if pl.Attribute != field {
			t.Errorf("expected field=%s, got %s", field, pl.Attribute)
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
	if !errors.Is(err, E.ErrInvalid) {
		t.Errorf("expected err InvalidProxyLabel, got %s", err)
	}
}

func TestHeaderProxyLabelValid(t *testing.T) {
	alias := "foo"
	field := "set_headers"
	v := `
	X-Custom-Header1: foo
	X-Custom-Header1: bar
	X-Custom-Header2: baz
	`
	h := make(http.Header, 0)
	h.Set("X-Custom-Header1", "foo")
	h.Add("X-Custom-Header1", "bar")
	h.Set("X-Custom-Header2", "baz")

	pl, err := ParseLabel(makeLabel(NSProxy, alias, field), v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err)
	}
	if pl.Target != alias {
		t.Errorf("expected alias=%s, got %s", alias, pl.Target)
	}
	if pl.Attribute != field {
		t.Errorf("expected field=%s, got %s", field, pl.Attribute)
	}
	hGot, ok := pl.Value.(http.Header)
	if !ok {
		t.Error("value is not http.Header")
		return
	}
	for k, vWant := range h {
		vGot := hGot[k]
		if !reflect.DeepEqual(vGot, vWant) {
			t.Errorf("expected %s=%q, got %q", k, vWant, vGot)
		}
	}
}

func TestHeaderProxyLabelInvalid(t *testing.T) {
	alias := "foo"
	field := "set_headers"
	tests := []string{
		"X-Custom-Header1 = bar",
		"X-Custom-Header1",
	}

	for _, v := range tests {
		_, err := ParseLabel(makeLabel(NSProxy, alias, field), v)
		if !errors.Is(err, E.ErrInvalid) {
			t.Errorf("expected err InvalidProxyLabel for %q, got %v", v, err)
		}
	}
}

func TestCommaSepProxyLabelSingle(t *testing.T) {
	alias := "foo"
	field := "hide_headers"
	v := "X-Custom-Header1"
	pl, err := ParseLabel(makeLabel(NSProxy, alias, field), v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err)
	}
	if pl.Target != alias {
		t.Errorf("expected alias=%s, got %s", alias, pl.Target)
	}
	if pl.Attribute != field {
		t.Errorf("expected field=%s, got %s", field, pl.Attribute)
	}
	sGot, ok := pl.Value.([]string)
	sWant := []string{"X-Custom-Header1"}
	if !ok {
		t.Error("value is not []string")
	}
	if !reflect.DeepEqual(sGot, sWant) {
		t.Errorf("expected %q, got %q", sWant, sGot)
	}
}

func TestCommaSepProxyLabelMulti(t *testing.T) {
	alias := "foo"
	field := "hide_headers"
	v := "X-Custom-Header1, X-Custom-Header2,X-Custom-Header3"
	pl, err := ParseLabel(makeLabel(NSProxy, alias, field), v)
	if err.IsNotNil() {
		t.Errorf("expected err=nil, got %s", err)
	}
	if pl.Target != alias {
		t.Errorf("expected alias=%s, got %s", alias, pl.Target)
	}
	if pl.Attribute != field {
		t.Errorf("expected field=%s, got %s", field, pl.Attribute)
	}
	sGot, ok := pl.Value.([]string)
	sWant := []string{"X-Custom-Header1", "X-Custom-Header2", "X-Custom-Header3"}
	if !ok {
		t.Error("value is not []string")
	}
	if !reflect.DeepEqual(sGot, sWant) {
		t.Errorf("expected %q, got %q", sWant, sGot)
	}
}
