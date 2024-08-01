package fields

import (
	F "github.com/yusing/go-proxy/utils/functional"
	E "github.com/yusing/go-proxy/error"
)

type PathMode struct{ F.Stringable }

func NewPathMode(pm string) (PathMode, E.NestedError) {
	switch pm {
	case "", "forward":
		return PathMode{F.NewStringable(pm)}, E.Nil()
	default:
		return PathMode{}, E.Invalid("path mode", pm)
	}
}

func (p PathMode) IsRemove() bool {
	return p.String() == ""
}

func (p PathMode) IsForward() bool {
	return p.String() == "forward"
}
