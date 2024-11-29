package utils

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/utils/strutils/ansi"
)

func init() {
	if common.IsTest {
		os.Args = append([]string{os.Args[0], "-test.v"}, os.Args[1:]...)
	}
}

func IgnoreError[Result any](r Result, _ error) Result {
	return r
}

func ExpectNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil && !reflect.ValueOf(err).IsNil() {
		t.Errorf("expected err=nil, got %v", err)
		t.FailNow()
	}
}

func ExpectError(t *testing.T, expected error, err error) {
	t.Helper()
	if !errors.Is(err, expected) {
		t.Errorf("expected err %s, got %v", expected, err)
		t.FailNow()
	}
}

func ExpectError2(t *testing.T, input any, expected error, err error) {
	t.Helper()
	if !errors.Is(err, expected) {
		t.Errorf("%v: expected err %s, got %v", input, expected, err)
		t.FailNow()
	}
}

func ExpectErrorT[T error](t *testing.T, err error) {
	t.Helper()
	var errAs T
	if !errors.As(err, &errAs) {
		t.Errorf("expected err %T, got %v", errAs, err)
		t.FailNow()
	}
}

func ExpectEqual[T comparable](t *testing.T, got T, want T) {
	t.Helper()
	if got != want {
		t.Errorf("expected:\n%v, got\n%v", want, got)
		t.FailNow()
	}
}

func ExpectStrEqual(t *testing.T, got string, want string) {
	t.Helper()
	got = ansi.StripANSI(got)
	if got != want {
		t.Errorf("expected:\n%v, got\n%v", want, got)
		t.FailNow()
	}
}

func ExpectEqualAny[T comparable](t *testing.T, got T, wants []T) {
	t.Helper()
	for _, want := range wants {
		if got == want {
			return
		}
	}
	t.Errorf("expected any of:\n%v, got\n%v", wants, got)
	t.FailNow()
}

func ExpectDeepEqual[T any](t *testing.T, got T, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected:\n%v, got\n%v", want, got)
		t.FailNow()
	}
}

func ExpectTrue(t *testing.T, got bool) {
	t.Helper()
	if !got {
		t.Error("expected true")
		t.FailNow()
	}
}

func ExpectFalse(t *testing.T, got bool) {
	t.Helper()
	if got {
		t.Error("expected false")
		t.FailNow()
	}
}

func ExpectType[T any](t *testing.T, got any) (_ T) {
	t.Helper()
	_, ok := got.(T)
	if !ok {
		t.Fatalf("expected type %s, got %s", reflect.TypeFor[T](), reflect.TypeOf(got).Elem())
		return
	}
	return got.(T)
}
