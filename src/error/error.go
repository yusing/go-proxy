package error

import (
	"errors"
	"fmt"
	"strings"
)

type (
	// NestedError is an error with an inner error
	// and a list of extra nested errors.
	//
	// It is designed to be non nil.
	//
	// You can use it to join multiple errors,
	// or to set a inner reason for a nested error.
	//
	// When a method returns both valid values and errors,
	// You should return (Slice/Map, NestedError).
	// Caller then should handle the nested error,
	// and continue with the valid values.
	NestedError struct {
		subject string
		err     error // can be nil
		extras  []NestedError
	}
)

func Nil() NestedError { return NestedError{} }

func From(err error) NestedError {
	switch err := err.(type) {
	case nil:
		return Nil()
	case NestedError:
		return err
	default:
		return NestedError{err: err}
	}
}

// Check is a helper function that
// convert (T, error) to (T, NestedError).
func Check[T any](obj T, err error) (T, NestedError) {
	return obj, From(err)
}

func Join(message string, err ...error) NestedError {
	extras := make([]NestedError, 0, len(err))
	nErr := 0
	for _, e := range err {
		if err == nil {
			continue
		}
		extras = append(extras, From(e))
		nErr += 1
	}
	if nErr == 0 {
		return Nil()
	}
	return NestedError{
		err:    errors.New(message),
		extras: extras,
	}
}

func (ne NestedError) Error() string {
	var buf strings.Builder
	ne.writeToSB(&buf, 0, "")
	return buf.String()
}

func (ne NestedError) Is(err error) bool {
	return errors.Is(ne.err, err)
}

func (ne NestedError) With(s any) NestedError {
	var msg string
	switch ss := s.(type) {
	case nil:
		return ne
	case error:
		return ne.withError(ss)
	case string:
		msg = ss
	case fmt.Stringer:
		msg = ss.String()
	default:
		msg = fmt.Sprint(s)
	}
	return ne.withError(errors.New(msg))
}

func (ne NestedError) Extraf(format string, args ...any) NestedError {
	return ne.With(fmt.Errorf(format, args...))
}

func (ne NestedError) Subject(s any) NestedError {
	switch ss := s.(type) {
	case string:
		ne.subject = ss
	case fmt.Stringer:
		ne.subject = ss.String()
	default:
		ne.subject = fmt.Sprint(s)
	}
	return ne
}

func (ne NestedError) Subjectf(format string, args ...any) NestedError {
	if strings.Contains(format, "%q") {
		panic("Subjectf format should not contain %q")
	}
	if strings.Contains(format, "%w") {
		panic("Subjectf format should not contain %w")
	}
	ne.subject = fmt.Sprintf(format, args...)
	return ne
}

func (ne NestedError) IsNil() bool {
	return ne.err == nil
}

func (ne NestedError) IsNotNil() bool {
	return ne.err != nil
}

func errorf(format string, args ...any) NestedError {
	return From(fmt.Errorf(format, args...))
}

func (ne NestedError) withError(err error) NestedError {
	ne.extras = append(ne.extras, From(err))
	return ne
}

func (ne *NestedError) writeToSB(sb *strings.Builder, level int, prefix string) {
	ne.writeIndents(sb, level)
	sb.WriteString(prefix)

	if ne.err != nil {
		sb.WriteString(ne.err.Error())
	}
	if ne.subject != "" {
		if ne.err != nil {
			sb.WriteString(fmt.Sprintf(" for %q", ne.subject))
		} else {
			sb.WriteString(fmt.Sprint(ne.subject))
		}
	}
	if len(ne.extras) > 0 {
		sb.WriteRune(':')
		for _, extra := range ne.extras {
			sb.WriteRune('\n')
			extra.writeToSB(sb, level+1, "- ")
		}
	}
}

func (ne *NestedError) writeIndents(sb *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		sb.WriteString("  ")
	}
}
