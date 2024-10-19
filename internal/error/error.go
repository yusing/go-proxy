package error

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type (
	NestedError     = *NestedErrorImpl
	NestedErrorImpl struct {
		subject string
		err     error
		extras  []NestedErrorImpl
	}
	JSONNestedError struct {
		Subject string            `json:"subject"`
		Err     string            `json:"error"`
		Extras  []JSONNestedError `json:"extras,omitempty"`
	}
)

func From(err error) NestedError {
	if IsNil(err) {
		return nil
	}
	return &NestedErrorImpl{err: err}
}

func FromJSON(data []byte) (NestedError, bool) {
	var j JSONNestedError
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, false
	}
	if j.Err == "" {
		return nil, false
	}
	extras := make([]NestedErrorImpl, len(j.Extras))
	for i, e := range j.Extras {
		extra, ok := fromJSONObject(e)
		if !ok {
			return nil, false
		}
		extras[i] = *extra
	}
	return &NestedErrorImpl{
		subject: j.Subject,
		err:     errors.New(j.Err),
		extras:  extras,
	}, true
}

// Check is a helper function that
// convert (T, error) to (T, NestedError).
func Check[T any](obj T, err error) (T, NestedError) {
	return obj, From(err)
}

func Join(message string, err ...NestedError) NestedError {
	extras := make([]NestedErrorImpl, len(err))
	nErr := 0
	for i, e := range err {
		if e == nil {
			continue
		}
		extras[i] = *e
		nErr++
	}
	if nErr == 0 {
		return nil
	}
	return &NestedErrorImpl{
		err:    errors.New(message),
		extras: extras,
	}
}

func JoinE(message string, err ...error) NestedError {
	b := NewBuilder("%s", message)
	for _, e := range err {
		b.AddE(e)
	}
	return b.Build()
}

func IsNil(err error) bool {
	return err == nil
}

func IsNotNil(err error) bool {
	return err != nil
}

func (ne NestedError) String() string {
	var buf strings.Builder
	ne.writeToSB(&buf, 0, "")
	return buf.String()
}

func (ne NestedError) Is(err error) bool {
	if ne == nil {
		return err == nil
	}
	// return errors.Is(ne.err, err)
	if errors.Is(ne.err, err) {
		return true
	}
	for _, e := range ne.extras {
		if e.Is(err) {
			return true
		}
	}
	return false
}

func (ne NestedError) IsNot(err error) bool {
	return !ne.Is(err)
}

func (ne NestedError) Error() error {
	if ne == nil {
		return nil
	}
	return ne.buildError(0, "")
}

func (ne NestedError) With(s any) NestedError {
	if ne == nil {
		return ne
	}
	var msg string
	switch ss := s.(type) {
	case nil:
		return ne
	case *NestedErrorImpl:
		if len(ss.extras) == 1 {
			ne.extras = append(ne.extras, ss.extras[0])
			return ne
		}
		return ne.withError(ss)
	case error:
		return ne.withError(From(ss))
	case string:
		msg = ss
	case fmt.Stringer:
		return ne.appendMsg(ss.String())
	default:
		return ne.appendMsg(fmt.Sprint(s))
	}
	return ne.withError(From(errors.New(msg)))
}

func (ne NestedError) Extraf(format string, args ...any) NestedError {
	return ne.With(errorf(format, args...))
}

func (ne NestedError) Subject(s any, sep ...string) NestedError {
	if ne == nil {
		return ne
	}
	var subject string
	switch ss := s.(type) {
	case string:
		subject = ss
	case fmt.Stringer:
		subject = ss.String()
	default:
		subject = fmt.Sprint(s)
	}
	switch {
	case ne.subject == "":
		ne.subject = subject
	case len(sep) > 0:
		ne.subject = fmt.Sprintf("%s%s%s", subject, sep[0], ne.subject)
	default:
		ne.subject = fmt.Sprintf("%s > %s", subject, ne.subject)
	}
	return ne
}

func (ne NestedError) Subjectf(format string, args ...any) NestedError {
	if ne == nil {
		return ne
	}
	return ne.Subject(fmt.Sprintf(format, args...))
}

func (ne NestedError) JSONObject() JSONNestedError {
	extras := make([]JSONNestedError, len(ne.extras))
	for i, e := range ne.extras {
		extras[i] = e.JSONObject()
	}
	return JSONNestedError{
		Subject: ne.subject,
		Err:     ne.err.Error(),
		Extras:  extras,
	}
}

func (ne NestedError) JSON() []byte {
	b, err := json.MarshalIndent(ne.JSONObject(), "", "  ")
	if err != nil {
		panic(err)
	}
	return b
}

func (ne NestedError) NoError() bool {
	return ne == nil
}

func (ne NestedError) HasError() bool {
	return ne != nil
}

func errorf(format string, args ...any) NestedError {
	return From(fmt.Errorf(format, args...))
}

func fromJSONObject(obj JSONNestedError) (NestedError, bool) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, false
	}
	return FromJSON(data)
}

func (ne NestedError) withError(err NestedError) NestedError {
	if ne != nil && err != nil {
		ne.extras = append(ne.extras, *err)
	}
	return ne
}

func (ne NestedError) appendMsg(msg string) NestedError {
	if ne == nil {
		return nil
	}
	ne.err = fmt.Errorf("%w %s", ne.err, msg)
	return ne
}

func (ne NestedError) writeToSB(sb *strings.Builder, level int, prefix string) {
	for range level {
		sb.WriteString("  ")
	}
	sb.WriteString(prefix)

	if ne.NoError() {
		sb.WriteString("nil")
		return
	}

	if ne.subject != "" {
		sb.WriteString(ne.subject)
		sb.WriteRune(' ')
	}
	sb.WriteString(ne.err.Error())
	if len(ne.extras) > 0 {
		sb.WriteRune(':')
		for _, extra := range ne.extras {
			sb.WriteRune('\n')
			extra.writeToSB(sb, level+1, "- ")
		}
	}
}

func (ne NestedError) buildError(level int, prefix string) error {
	var res error
	var sb strings.Builder

	for range level {
		sb.WriteString("  ")
	}
	sb.WriteString(prefix)

	if ne.NoError() {
		sb.WriteString("nil")
		return errors.New(sb.String())
	}

	res = fmt.Errorf("%s%w", sb.String(), ne.err)
	sb.Reset()

	if ne.subject != "" {
		sb.WriteString(fmt.Sprintf(" for %q", ne.subject))
	}
	if len(ne.extras) > 0 {
		sb.WriteRune(':')
		res = fmt.Errorf("%w%s", res, sb.String())
		for _, extra := range ne.extras {
			res = errors.Join(res, extra.buildError(level+1, "- "))
		}
	} else {
		res = fmt.Errorf("%w%s", res, sb.String())
	}
	return res
}
