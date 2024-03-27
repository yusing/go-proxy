package main

import (
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

func (ef *NestedError) Error() string {
	var buf strings.Builder
	ef.writeToSB(&buf, "")
	return buf.String()
}

func (ef *NestedError) HasInner() bool {
	return ef.inner != nil
}

func (ef *NestedError) HasExtras() bool {
	return len(ef.extras) > 0
}

func (ef *NestedError) With(inner error) NestedErrorLike {
	ef.Lock()
	defer ef.Unlock()

	var in *NestedError

	switch t := inner.(type) {
	case NestedErrorLike:
		in = t.copy()
	default:
		in = &NestedError{extras: []string{t.Error()}}
	}
	if ef.inner == nil {
		ef.inner = in
	} else {
		ef.inner.ExtraError(in)
	}
	root := ef
	for root.inner != nil {
		root.inner.level = root.level + 1
		root = root.inner
	}
	return ef
}

func (ef *NestedError) addLevel(level int) NestedErrorLike {
	ef.level += level
	if ef.inner != nil {
		ef.inner.addLevel(level)
	}
	return ef
}

func (ef *NestedError) copy() *NestedError {
	var inner *NestedError
	if ef.inner != nil {
		inner = ef.inner.copy()
	}
	return &NestedError{
		subject: ef.subject,
		message: ef.message,
		extras:  ef.extras,
		inner:   inner,
		level:   ef.level,
	}
}

func (ef *NestedError) writeIndents(sb *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		sb.WriteString("  ")
	}
}

func (ef *NestedError) writeToSB(sb *strings.Builder, prefix string) {
	ef.writeIndents(sb, ef.level)
	sb.WriteString(prefix)

	if ef.subject != "" {
		sb.WriteRune('"')
		sb.WriteString(ef.subject)
		sb.WriteRune('"')
		if ef.message != "" {
			sb.WriteString(":\n")
		} else {
			sb.WriteRune('\n')
		}
	}
	if ef.message != "" {
		ef.writeIndents(sb, ef.level)
		sb.WriteString(ef.message)
		sb.WriteRune('\n')
	}
	for _, l := range ef.extras {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		ef.writeIndents(sb, ef.level)
		sb.WriteString("- ")
		sb.WriteString(l)
		sb.WriteRune('\n')
	}
	if ef.inner != nil {
		ef.inner.writeToSB(sb, "- ")
	}
}
