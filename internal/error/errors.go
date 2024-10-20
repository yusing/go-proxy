package error

import (
	stderrors "errors"
	"fmt"
	"reflect"
)

var (
	ErrFailure      = stderrors.New("failed")
	ErrInvalid      = stderrors.New("invalid")
	ErrUnsupported  = stderrors.New("unsupported")
	ErrUnexpected   = stderrors.New("unexpected")
	ErrNotExists    = stderrors.New("does not exist")
	ErrMissing      = stderrors.New("missing")
	ErrDuplicated   = stderrors.New("duplicated")
	ErrOutOfRange   = stderrors.New("out of range")
	ErrTypeError    = stderrors.New("type error")
	ErrTypeMismatch = stderrors.New("type mismatch")
	ErrPanicRecv    = stderrors.New("panic recovered from")
)

const fmtSubjectWhat = "%w %v: %q"

func Failure(what string) Error {
	return errorf("%s %w", what, ErrFailure)
}

func FailedWhy(what string, why string) Error {
	return Failure(what).With(why)
}

func FailWith(what string, err any) Error {
	return Failure(what).With(err)
}

func Invalid(subject, what any) Error {
	return errorf(fmtSubjectWhat, ErrInvalid, subject, what)
}

func Unsupported(subject, what any) Error {
	return errorf(fmtSubjectWhat, ErrUnsupported, subject, what)
}

func Unexpected(subject, what any) Error {
	return errorf(fmtSubjectWhat, ErrUnexpected, subject, what)
}

func UnexpectedError(err error) Error {
	return errorf("%w error: %w", ErrUnexpected, err)
}

func NotExist(subject, what any) Error {
	return errorf("%v %w: %v", subject, ErrNotExists, what)
}

func Missing(subject any) Error {
	return errorf("%w %v", ErrMissing, subject)
}

func Duplicated(subject, what any) Error {
	return errorf("%w %v: %v", ErrDuplicated, subject, what)
}

func OutOfRange(subject any, value any) Error {
	return errorf("%v %w: %v", subject, ErrOutOfRange, value)
}

func TypeError(subject any, from, to reflect.Type) Error {
	return errorf("%v %w: %s -> %s\n", subject, ErrTypeError, from, to)
}

func TypeError2(subject any, from, to reflect.Value) Error {
	return TypeError(subject, from.Type(), to.Type())
}

func TypeMismatch[Expect any](value any) Error {
	return errorf("%w: expect %s got %T", ErrTypeMismatch, reflect.TypeFor[Expect](), value)
}

func PanicRecv(format string, args ...any) Error {
	return errorf("%w %s", ErrPanicRecv, fmt.Sprintf(format, args...))
}
