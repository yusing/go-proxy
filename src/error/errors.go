package error

import (
	stderrors "errors"
)

var (
	ErrFailure      = stderrors.New("failed")
	ErrInvalid      = stderrors.New("invalid")
	ErrUnsupported  = stderrors.New("unsupported")
	ErrUnexpected   = stderrors.New("unexpected")
	ErrNotExists    = stderrors.New("does not exist")
	ErrMissing      = stderrors.New("missing")
	ErrAlreadyExist = stderrors.New("already exist")
	ErrOutOfRange   = stderrors.New("out of range")
)

const fmtSubjectWhat = "%w %v: %q"

func Failure(what string) NestedError {
	return errorf("%s %w", what, ErrFailure)
}

func FailedWhy(what string, why string) NestedError {
	return Failure(what).With(why)
}

func FailWith(what string, err any) NestedError {
	return Failure(what).With(err)
}

func Invalid(subject, what any) NestedError {
	return errorf(fmtSubjectWhat, ErrInvalid, subject, what)
}

func Unsupported(subject, what any) NestedError {
	return errorf(fmtSubjectWhat, ErrUnsupported, subject, what)
}

func Unexpected(subject, what any) NestedError {
	return errorf(fmtSubjectWhat, ErrUnexpected, subject, what)
}

func UnexpectedError(err error) NestedError {
	return errorf("%w error: %w", ErrUnexpected, err)
}

func NotExist(subject, what any) NestedError {
	return errorf("%v %w: %v", subject, ErrNotExists, what)
}

func Missing(subject any) NestedError {
	return errorf("%w %v", ErrMissing, subject)
}

func AlreadyExist(subject, what any) NestedError {
	return errorf("%v %w: %v", subject, ErrAlreadyExist, what)
}

func OutOfRange(subject string, value any) NestedError {
	return errorf("%v %w: %v", subject, ErrOutOfRange, value)
}
