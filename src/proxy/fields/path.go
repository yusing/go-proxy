package fields

import (
	E "github.com/yusing/go-proxy/error"
)

type Path string

func NewPath(s string) (Path, E.NestedError) {
	if s == "" || s[0] == '/' {
		return Path(s), E.Nil()
	}
	return "", E.Invalid("path", s).With("must be empty or start with '/'")
}
