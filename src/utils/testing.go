package utils

import (
	"reflect"
	"testing"

	E "github.com/yusing/go-proxy/error"
)

func ExpectNoError(t *testing.T, err error) {
	t.Helper()
	var noError bool
	switch t := err.(type) {
	case E.NestedError:
		noError = t.NoError()
	default:
		noError = err == nil
	}
	if !noError {
		t.Errorf("expected err=nil, got %s", err.Error())
	}
}

func ExpectEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected:\n%v, got\n%v", want, got)
	}
}

func ExpectTrue(t *testing.T, got bool) {
	t.Helper()
	if !got {
		t.Errorf("expected true, got false")
	}
}

func ExpectFalse(t *testing.T, got bool) {
	t.Helper()
	if got {
		t.Errorf("expected false, got true")
	}
}

func ExpectType[T any](t *testing.T, got any) T {
	t.Helper()
	tExpect := reflect.TypeFor[T]()
	_, ok := got.(T)
	if !ok {
		t.Errorf("expected type %T, got %T", tExpect, got)
	}
	return got.(T)
}
