package error

import (
	"testing"
)

func AssertEq[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("expected:\n%v, got\n%v", want, got)
	}
}

func TestErrorIs(t *testing.T) {
	AssertEq(t, Failure("foo").Is(ErrFailure), true)
	AssertEq(t, Failure("foo").With("bar").Is(ErrFailure), true)
	AssertEq(t, Failure("foo").With("bar").Is(ErrInvalid), false)
	AssertEq(t, Failure("foo").With("bar").With("baz").Is(ErrInvalid), false)

	AssertEq(t, Invalid("foo", "bar").Is(ErrInvalid), true)
	AssertEq(t, Invalid("foo", "bar").Is(ErrFailure), false)

	AssertEq(t, Nil().Is(nil), true)
	AssertEq(t, Nil().Is(ErrInvalid), false)
	AssertEq(t, Invalid("foo", "bar").Is(nil), false)
}

func TestNil(t *testing.T) {
	AssertEq(t, Nil().IsNil(), true)
	AssertEq(t, Nil().IsNotNil(), false)
	AssertEq(t, Nil().Error(), "nil")
}

func TestErrorSimple(t *testing.T) {
	ne := Failure("foo bar")
	AssertEq(t, ne.Error(), "foo bar failed")
	ne = ne.Subject("baz")
	AssertEq(t, ne.Error(), "foo bar failed for \"baz\"")
}

func TestErrorWith(t *testing.T) {
	ne := Failure("foo").With("bar").With("baz")
	AssertEq(t, ne.Error(), "foo failed:\n  - bar\n  - baz")
}

func TestErrorNested(t *testing.T) {
	inner := Failure("inner").
		With("1").
		With("1")
	inner2 := Failure("inner2").
		Subject("action 2").
		With("2").
		With("2")
	inner3 := Failure("inner3").
		Subject("action 3").
		With("3").
		With("3")
	ne := Failure("foo").
		With("bar").
		With("baz").
		With(inner).
		With(inner.With(inner2.With(inner3)))
	want :=
		`foo failed:
  - bar
  - baz
  - inner failed:
    - 1
    - 1
  - inner failed:
    - 1
    - 1
    - inner2 failed for "action 2":
      - 2
      - 2
      - inner3 failed for "action 3":
        - 3
        - 3`
	AssertEq(t, ne.Error(), want)
}
