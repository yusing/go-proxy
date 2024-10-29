package error

import (
	"errors"
	"fmt"
	"strings"
)

type nestedError struct {
	Err    error   `json:"err"`
	Extras []error `json:"extras"`
}

func (err nestedError) Subject(subject string) Error {
	if err.Err == nil {
		err.Err = newError(subject)
	} else {
		err.Err = PrependSubject(subject, err.Err)
	}
	return &err
}

func (err *nestedError) Subjectf(format string, args ...any) Error {
	if len(args) > 0 {
		return err.Subject(fmt.Sprintf(format, args...))
	}
	return err.Subject(format)
}

func (err nestedError) With(extra error) Error {
	if extra != nil {
		err.Extras = append(err.Extras, extra)
	}
	return &err
}

func (err nestedError) Withf(format string, args ...any) Error {
	if len(args) > 0 {
		err.Extras = append(err.Extras, fmt.Errorf(format, args...))
	} else {
		err.Extras = append(err.Extras, newError(format))
	}
	return &err
}

func (err *nestedError) Unwrap() []error {
	if err.Err == nil {
		if len(err.Extras) == 0 {
			return nil
		}
		return err.Extras
	}
	return append([]error{err.Err}, err.Extras...)
}

func (err *nestedError) Is(other error) bool {
	if errors.Is(err.Err, other) {
		return true
	}
	for _, e := range err.Extras {
		if errors.Is(e, other) {
			return true
		}
	}
	return false
}

func (err *nestedError) Error() string {
	return buildError(err, 0)
}

//go:inline
func makeLine(err string, level int) string {
	const bulletPrefix = "• "
	const spaces = "                "

	if level == 0 {
		return err
	}
	return spaces[:2*level] + bulletPrefix + err
}

func makeLines(errs []error, level int) []string {
	if len(errs) == 0 {
		return nil
	}
	lines := make([]string, 0, len(errs))
	for _, err := range errs {
		switch err := err.(type) {
		case *nestedError:
			if err.Err != nil {
				lines = append(lines, makeLine(err.Err.Error(), level))
			}
			if extras := makeLines(err.Extras, level+1); len(extras) > 0 {
				lines = append(lines, extras...)
			}
		default:
			lines = append(lines, makeLine(err.Error(), level))
		}
	}
	return lines
}

func buildError(err error, level int) string {
	switch err := err.(type) {
	case nil:
		return makeLine("<nil>", level)
	case *nestedError:
		lines := make([]string, 0, 1+len(err.Extras))
		if err.Err != nil {
			lines = append(lines, makeLine(err.Err.Error(), level))
		}
		if extras := makeLines(err.Extras, level+1); len(extras) > 0 {
			lines = append(lines, extras...)
		}
		return strings.Join(lines, "\n")
	default:
		return makeLine(err.Error(), level)
	}
}
