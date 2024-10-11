package error_test

import (
	"errors"
	"testing"

	. "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestErrorIs(t *testing.T) {
	ExpectTrue(t, Failure("foo").Is(ErrFailure))
	ExpectTrue(t, Failure("foo").With("bar").Is(ErrFailure))
	ExpectFalse(t, Failure("foo").With("bar").Is(ErrInvalid))
	ExpectFalse(t, Failure("foo").With("bar").With("baz").Is(ErrInvalid))

	ExpectTrue(t, Invalid("foo", "bar").Is(ErrInvalid))
	ExpectFalse(t, Invalid("foo", "bar").Is(ErrFailure))

	ExpectFalse(t, Invalid("foo", "bar").Is(nil))

	ExpectTrue(t, errors.Is(Failure("foo").Error(), ErrFailure))
	ExpectTrue(t, errors.Is(Failure("foo").With(Invalid("bar", "baz")).Error(), ErrInvalid))
	ExpectTrue(t, errors.Is(Failure("foo").With(Invalid("bar", "baz")).Error(), ErrFailure))
	ExpectFalse(t, errors.Is(Failure("foo").With(Invalid("bar", "baz")).Error(), ErrNotExists))
}

func TestErrorNestedIs(t *testing.T) {
	var err NestedError
	ExpectTrue(t, err.Is(nil))

	err = Failure("some reason")
	ExpectTrue(t, err.Is(ErrFailure))
	ExpectFalse(t, err.Is(ErrDuplicated))

	err.With(Duplicated("something", ""))
	ExpectTrue(t, err.Is(ErrFailure))
	ExpectTrue(t, err.Is(ErrDuplicated))
	ExpectFalse(t, err.Is(ErrInvalid))
}

func TestIsNil(t *testing.T) {
	var err NestedError
	ExpectTrue(t, err.Is(nil))
	ExpectFalse(t, err.HasError())
	ExpectTrue(t, err == nil)
	ExpectTrue(t, err.NoError())

	eb := NewBuilder("")
	returnNil := func() error {
		return eb.Build().Error()
	}
	ExpectTrue(t, IsNil(returnNil()))
	ExpectTrue(t, returnNil() == nil)

	ExpectTrue(t, (err.
		Subject("any").
		With("something").
		Extraf("foo %s", "bar")) == nil)
}

func TestErrorSimple(t *testing.T) {
	ne := Failure("foo bar")
	ExpectEqual(t, ne.String(), "foo bar failed")
	ne = ne.Subject("baz")
	ExpectEqual(t, ne.String(), "foo bar failed for \"baz\"")
}

func TestErrorWith(t *testing.T) {
	ne := Failure("foo").With("bar").With("baz")
	ExpectEqual(t, ne.String(), "foo failed:\n  - bar\n  - baz")
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
	want := `foo failed:
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
	ExpectEqual(t, ne.String(), want)
	ExpectEqual(t, ne.Error().Error(), want)
}
