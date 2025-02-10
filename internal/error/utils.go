package err

import (
	"fmt"
)

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

func Wrap(err error, message ...string) Error {
	if len(message) == 0 || message[0] == "" {
		return From(err)
	}
	return Errorf("%w: %s", err, message[0])
}

func From(err error) Error {
	if err == nil {
		return nil
	}
	//nolint:errorlint
	switch err := err.(type) {
	case *baseError:
		return err
	case *nestedError:
		return err
	}
	return &baseError{err}
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
	errs := make([]error, n)
	i := 0
	for _, err := range errors {
		if err != nil {
			errs[i] = err
			i++
		}
	}
	return &nestedError{Extras: errs}
}

func Collect[T any, Err error, Arg any, Func func(Arg) (T, Err)](eb *Builder, fn Func, arg Arg) T {
	result, err := fn(arg)
	eb.Add(err)
	return result
}
