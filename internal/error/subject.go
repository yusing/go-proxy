package err

import (
	"strings"

	"github.com/yusing/go-proxy/internal/utils/strutils/ansi"
)

//nolint:errname
type withSubject struct {
	Subjects []string `json:"subjects"`
	Err      error    `json:"err"`
}

const subjectSep = " > "

func highlight(subject string) string {
	return ansi.HighlightRed + subject + ansi.Reset
}

func PrependSubject(subject string, err error) error {
	if err == nil {
		return nil
	}

	//nolint:errorlint
	switch err := err.(type) {
	case *withSubject:
		return err.Prepend(subject)
	case Error:
		return err.Subject(subject)
	}
	return &withSubject{[]string{subject}, err}
}

func (err *withSubject) Prepend(subject string) *withSubject {
	clone := *err
	if subject != "" {
		switch subject[0] {
		case '[', '(', '{':
			clone.Subjects[len(clone.Subjects)-1] += subject
		default:
			clone.Subjects = append(clone.Subjects, subject)
		}
	}
	return &clone
}

func (err *withSubject) Is(other error) bool {
	return err.Err == other
}

func (err *withSubject) Unwrap() error {
	return err.Err
}

func (err *withSubject) Error() string {
	// subject is in reversed order
	n := len(err.Subjects)
	size := 0
	errStr := err.Err.Error()
	var sb strings.Builder
	for _, s := range err.Subjects {
		size += len(s)
	}
	sb.Grow(size + 2 + n*len(subjectSep) + len(errStr))

	for i := n - 1; i > 0; i-- {
		sb.WriteString(err.Subjects[i])
		sb.WriteString(subjectSep)
	}
	sb.WriteString(highlight(err.Subjects[0]))
	sb.WriteString(": ")
	sb.WriteString(errStr)
	return sb.String()
}
