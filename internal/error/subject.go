package err

import (
	"strings"

	"github.com/yusing/go-proxy/internal/utils/strutils/ansi"
)

//nolint:errname
type withSubject struct {
	Subject string `json:"subject"`
	Err     error  `json:"err"`
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
	return &withSubject{subject, err}
}

func (err *withSubject) Prepend(subject string) *withSubject {
	clone := *err
	if subject != "" {
		clone.Subject = subject + subjectSep + clone.Subject
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
	subjects := strings.Split(err.Subject, subjectSep)
	subjects[len(subjects)-1] = highlight(subjects[len(subjects)-1])
	return strings.Join(subjects, subjectSep) + ": " + err.Err.Error()
}
