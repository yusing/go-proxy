package error

import (
	"errors"
	"fmt"
	"strings"
	"sync"
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
	NestedError struct{ *nestedError }
	nestedError struct {
		neBase
		sync.Mutex
	}
	neBase struct {
		subject any
		err     error // can be nil
		extras  []neBase
		inner   *neBase // can be nil
		level   int
	}
)

func Nil() NestedError { return NestedError{} }

func From(err error) NestedError {
	if err == nil {
		return Nil()
	}
	return NestedError{&nestedError{neBase: *copyFrom(err)}}
}

// Check is a helper function that
// convert (T, error) to (T, NestedError).
func Check[T any](obj T, err error) (T, NestedError) {
	return obj, From(err)
}

func Join(message string, err ...error) NestedError {
	extras := make([]neBase, 0, len(err))
	nErr := 0
	for _, e := range err {
		if err == nil {
			continue
		}
		extras = append(extras, *copyFrom(e))
		nErr += 1
	}
	if nErr == 0 {
		return Nil()
	}
	return NestedError{&nestedError{
		neBase: neBase{
			err:    errors.New(message),
			extras: extras,
		},
	}}
}

func copyFrom(err error) *neBase {
	if err == nil {
		return nil
	}
	switch base := err.(type) {
	case *neBase:
		copy := *base
		return &copy
	}
	return &neBase{err: err}
}

func new(message ...string) NestedError {
	if len(message) == 0 {
		return From(nil)
	}
	return From(errors.New(strings.Join(message, " ")))
}

func errorf(format string, args ...any) NestedError {
	return From(fmt.Errorf(format, args...))
}

func (ne *neBase) Error() string {
	var buf strings.Builder
	ne.writeToSB(&buf, ne.level, "")
	return buf.String()
}

func (ne NestedError) ExtraError(err error) NestedError {
	if err != nil {
		ne.Lock()
		ne.extras = append(ne.extras, From(err).addLevel(ne.Level()+1))
		ne.Unlock()
	}
	return ne
}

func (ne NestedError) Extra(s string) NestedError {
	return ne.ExtraError(errors.New(s))
}

func (ne NestedError) ExtraAny(s any) NestedError {
	var msg string
	switch ss := s.(type) {
	case error:
		return ne.ExtraError(ss)
	case string:
		msg = ss
	case fmt.Stringer:
		msg = ss.String()
	default:
		msg = fmt.Sprint(s)
	}
	return ne.ExtraError(errors.New(msg))
}

func (ne NestedError) Extraf(format string, args ...any) NestedError {
	return ne.ExtraError(fmt.Errorf(format, args...))
}

func (ne NestedError) Subject(s any) NestedError {
	ne.subject = s
	return ne
}

func (ne NestedError) Subjectf(format string, args ...any) NestedError {
	if strings.Contains(format, "%q") {
		panic("Subjectf format should not contain %q")
	}
	if strings.Contains(format, "%w") {
		panic("Subjectf format should not contain %w")
	}
	return ne.Subject(fmt.Sprintf(format, args...))
}

func (ne NestedError) Level() int {
	return ne.level
}

func (ne *nestedError) IsNil() bool {
	return ne == nil
}

func (ne *nestedError) IsNotNil() bool {
	return ne != nil
}

func (ne NestedError) With(inner error) NestedError {
	ne.Lock()
	defer ne.Unlock()

	if ne.inner == nil {
		ne.inner = copyFrom(inner)
	} else {
		ne.ExtraError(inner)
	}

	root := &ne.neBase
	for root.inner != nil {
		root.inner.level = root.level + 1
		root = root.inner
	}
	return ne
}

func (ne *neBase) addLevel(level int) neBase {
	ret := *ne
	ret.level += level
	if ret.inner != nil {
		inner := ret.inner.addLevel(level)
		ret.inner = &inner
	}
	return ret
}

func (ne *neBase) writeToSB(sb *strings.Builder, level int, prefix string) {
	ne.writeIndents(sb, level)
	sb.WriteString(prefix)

	if ne.err != nil {
		sb.WriteString(ne.err.Error())
		sb.WriteRune(' ')
	}
	if ne.subject != nil {
		sb.WriteString(fmt.Sprintf("for %q", ne.subject))
	}
	if ne.inner != nil || len(ne.extras) > 0 {
		sb.WriteString(":\n")
	}
	level += 1
	for _, extra := range ne.extras {
		extra.writeToSB(sb, level, "- ")
		sb.WriteRune('\n')
	}
	if ne.inner != nil {
		ne.inner.writeToSB(sb, level, "- ")
	}
}

func (ne *neBase) writeIndents(sb *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		sb.WriteString("  ")
	}
}
