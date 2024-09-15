package fields

import (
	E "github.com/yusing/go-proxy/error"
)

type PathMode string

func NewPathMode(pm string) (PathMode, E.NestedError) {
	switch pm {
	case "", "forward":
		return PathMode(pm), E.Nil()
	default:
		return "", E.Invalid("path mode", pm)
	}
}

func (p PathMode) IsRemove() bool {
	return p == ""
}

func (p PathMode) IsForward() bool {
	return p == "forward"
}
