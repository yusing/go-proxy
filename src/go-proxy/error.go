package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

type NestedError struct {
	subject string
	message string
	extras  []string
	inner   *NestedError
	level   int

	sync.Mutex
}

type NestedErrorLike interface {
	Error() string
	Inner() NestedErrorLike
	Level() int
	HasInner() bool
	HasExtras() bool

	Extra(string) NestedErrorLike
	Extraf(string, ...any) NestedErrorLike
	ExtraError(error) NestedErrorLike
	Subject(string) NestedErrorLike
	Subjectf(string, ...any) NestedErrorLike
	With(error) NestedErrorLike

	addLevel(int) NestedErrorLike
	copy() *NestedError
}

func NewNestedError(message string) NestedErrorLike {
	return &NestedError{message: message, extras: make([]string, 0)}
}

func NewNestedErrorf(format string, args ...any) NestedErrorLike {
	return NewNestedError(fmt.Sprintf(format, args...))
}

func NewNestedErrorFrom(err error) NestedErrorLike {
	if err == nil {
		panic("cannot convert nil error to NestedError")
	}
	errUnwrap := errors.Unwrap(err)
	if errUnwrap != nil {
		return NewNestedErrorFrom(errUnwrap)
	}
	return NewNestedError(err.Error())
}

func (ne *NestedError) Extra(s string) NestedErrorLike {
	s = strings.TrimSpace(s)
	if s == "" {
		return ne
	}
	ne.Lock()
	defer ne.Unlock()
	ne.extras = append(ne.extras, s)
	return ne
}

func (ne *NestedError) Extraf(format string, args ...any) NestedErrorLike {
	return ne.Extra(fmt.Sprintf(format, args...))
}

func (ne *NestedError) ExtraError(e error) NestedErrorLike {
	switch t := e.(type) {
	case NestedErrorLike:
		extra := t.copy()
		extra.addLevel(ne.Level() + 1)
		e = extra
	}
	return ne.Extra(e.Error())
}

func (ne *NestedError) Subject(s string) NestedErrorLike {
	ne.subject = s
	return ne
}

func (ne *NestedError) Subjectf(format string, args ...any) NestedErrorLike {
	ne.subject = fmt.Sprintf(format, args...)
	return ne
}

func (ne *NestedError) Inner() NestedErrorLike {
	return ne.inner
}

func (ne *NestedError) Level() int {
	return ne.level
}

func (ne *NestedError) Error() string {
	var buf strings.Builder
	ne.writeToSB(&buf, ne.level, "")
	return buf.String()
}

func (ne *NestedError) HasInner() bool {
	return ne.inner != nil
}

func (ne *NestedError) HasExtras() bool {
	return len(ne.extras) > 0
}

func (ne *NestedError) With(inner error) NestedErrorLike {
	ne.Lock()
	defer ne.Unlock()

	var in *NestedError

	switch t := inner.(type) {
	case NestedErrorLike:
		in = t.copy()
	default:
		in = &NestedError{message: t.Error()}
	}
	if ne.inner == nil {
		ne.inner = in
	} else {
		ne.inner.ExtraError(in)
	}
	root := ne
	for root.inner != nil {
		root.inner.level = root.level + 1
		root = root.inner
	}
	return ne
}

func (ne *NestedError) addLevel(level int) NestedErrorLike {
	ne.level += level
	if ne.inner != nil {
		ne.inner.addLevel(level)
	}
	return ne
}

func (ne *NestedError) copy() *NestedError {
	var inner *NestedError
	if ne.inner != nil {
		inner = ne.inner.copy()
	}
	return &NestedError{
		subject: ne.subject,
		message: ne.message,
		extras:  ne.extras,
		inner:   inner,
	}
}

func (ne *NestedError) writeIndents(sb *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		sb.WriteString("  ")
	}
}

func (ne *NestedError) writeToSB(sb *strings.Builder, level int, prefix string) {
	ne.writeIndents(sb, level)
	sb.WriteString(prefix)

	if ne.subject != "" {
		sb.WriteString(ne.subject)
		if ne.message != "" {
			sb.WriteString(": ")
		}
	}
	if ne.message != "" {
		sb.WriteString(ne.message)
	}
	if ne.HasExtras() || ne.HasInner() {
		sb.WriteString(":\n")
	}
	level += 1
	for _, l := range ne.extras {
		if l == "" {
			continue
		}
		ne.writeIndents(sb, level)
		sb.WriteString("- ")
		sb.WriteString(l)
		sb.WriteRune('\n')
	}
	if ne.inner != nil {
		ne.inner.writeToSB(sb, level, "- ")
	}
}
