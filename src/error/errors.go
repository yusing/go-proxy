package error

import (
	stderrors "errors"
)

var (
	ErrFailure     = stderrors.New("failed")
	ErrInvalid     = stderrors.New("invalid")
	ErrUnsupported = stderrors.New("unsupported")
	ErrNotExists   = stderrors.New("does not exist")
	ErrDuplicated  = stderrors.New("duplicated")
)

func Failure(what string) NestedError {
	return errorf("%s %w", what, ErrFailure)
}

func FailureWhy(what string, why string) NestedError {
	return errorf("%s %w because %s", what, ErrFailure, why)
}

func Invalid(subject, what any) NestedError {
	return errorf("%w %v - %v", ErrInvalid, subject, what)
}

func Unsupported(subject, what any) NestedError {
	return errorf("%w %v - %v", ErrUnsupported, subject, what)
}

func NotExists(subject, what any) NestedError {
	return errorf("%s %v - %v", subject, ErrNotExists, what)
}

func Duplicated(subject, what any) NestedError {
	return errorf("%w %v: %v", ErrDuplicated, subject, what)
}
