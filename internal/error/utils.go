package error

import (
	"errors"
	"fmt"
)

var ErrInvalidErrorJson = errors.New("invalid error json")

func newError(message string) error {
	return errStr(message)
}

func New(message string) Error {
	if message == "" {
		return nil
	}
	return &baseError{newError(message)}
}

func Errorf(format string, args ...any) Error {
	return &baseError{fmt.Errorf(format, args...)}
}

func From(err error) Error {
	if err == nil {
		return nil
	}
	if err, ok := err.(Error); ok {
		return err
	}
	return &baseError{err}
}

func Must[T any](v T, err error) T {
	if err != nil {
		LogPanic("must failed", err)
	}
	return v
}

func Join(errors ...error) Error {
	n := 0
	for _, err := range errors {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	errs := make([]error, 0, n)
	for _, err := range errors {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return &nestedError{Extras: errs}
}

func Collect[T any, Err error, Arg any, Func func(Arg) (T, Err)](eb *Builder, fn Func, arg Arg) T {
	result, err := fn(arg)
	eb.Add(err)
	return result
}

func Collect2[T any, Err error, Arg1 any, Arg2 any, Func func(Arg1, Arg2) (T, Err)](eb *Builder, fn Func, arg1 Arg1, arg2 Arg2) T {
	result, err := fn(arg1, arg2)
	eb.Add(err)
	return result
}
