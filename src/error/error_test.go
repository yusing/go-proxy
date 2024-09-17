package error_test

import (
	"testing"

	. "github.com/yusing/go-proxy/error"
	. "github.com/yusing/go-proxy/utils"
)

func TestErrorIs(t *testing.T) {
	ExpectTrue(t, Failure("foo").Is(ErrFailure))
	ExpectTrue(t, Failure("foo").With("bar").Is(ErrFailure))
	ExpectFalse(t, Failure("foo").With("bar").Is(ErrInvalid))
	ExpectFalse(t, Failure("foo").With("bar").With("baz").Is(ErrInvalid))

	ExpectTrue(t, Invalid("foo", "bar").Is(ErrInvalid))
	ExpectFalse(t, Invalid("foo", "bar").Is(ErrFailure))

	ExpectTrue(t, Nil().Is(nil))
	ExpectFalse(t, Nil().Is(ErrInvalid))
	ExpectFalse(t, Invalid("foo", "bar").Is(nil))
}

func TestNil(t *testing.T) {
	ExpectTrue(t, Nil().NoError())
	ExpectFalse(t, Nil().HasError())
	ExpectEqual(t, Nil().Error(), "nil")
}

func TestErrorSimple(t *testing.T) {
	ne := Failure("foo bar")
	ExpectEqual(t, ne.Error(), "foo bar failed")
	ne = ne.Subject("baz")
	ExpectEqual(t, ne.Error(), "foo bar failed for \"baz\"")
}

func TestErrorWith(t *testing.T) {
	ne := Failure("foo").With("bar").With("baz")
	ExpectEqual(t, ne.Error(), "foo failed:\n  - bar\n  - baz")
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
	ExpectEqual(t, ne.Error(), want)
}
