package utils

import (
	"reflect"
	"testing"
)

func ExpectNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil && !reflect.ValueOf(err).IsNil() {
		t.Errorf("expected err=nil, got %s", err.Error())
	}
}

func ExpectEqual[T comparable](t *testing.T, got T, want T) {
	t.Helper()
	if got != want {
		t.Errorf("expected:\n%v, got\n%v", want, got)
	}
}

func ExpectDeepEqual[T any](t *testing.T, got T, want T) {
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
		t.Errorf("expected type %s, got %T", tExpect, got)
	}
	return got.(T)
}
