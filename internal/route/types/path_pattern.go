package types

import (
	"errors"
	"fmt"
	"regexp"

	E "github.com/yusing/go-proxy/internal/error"
)

type (
	PathPattern  string
	PathPatterns = []PathPattern
)

var pathPattern = regexp.MustCompile(`^(/[-\w./]*({\$\})?|((GET|POST|DELETE|PUT|HEAD|OPTION) /[-\w./]*({\$\})?))$`)

var (
	ErrEmptyPathPattern   = errors.New("path must not be empty")
	ErrInvalidPathPattern = errors.New("invalid path pattern")
)

func ValidatePathPattern(s string) (PathPattern, error) {
	if len(s) == 0 {
		return "", ErrEmptyPathPattern
	}
	if !pathPattern.MatchString(s) {
		return "", fmt.Errorf("%w %q", ErrInvalidPathPattern, s)
	}
	return PathPattern(s), nil
}

func ValidatePathPatterns(s []string) (PathPatterns, E.Error) {
	if len(s) == 0 {
		return nil, nil
	}
	errs := E.NewBuilder("invalid path patterns")
	pp := make(PathPatterns, len(s))
	for i, v := range s {
		pattern, err := ValidatePathPattern(v)
		if err != nil {
			errs.Add(err)
		} else {
			pp[i] = pattern
		}
	}
	return pp, errs.Error()
}
