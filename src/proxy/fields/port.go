package fields

import (
	"strconv"

	E "github.com/yusing/go-proxy/error"
)

type Port int

func NewPort(v string) (Port, E.NestedError) {
	p, err := strconv.Atoi(v)
	if err != nil {
		return ErrPort, E.From(err)
	}
	return NewPortInt(p)
}

func NewPortInt(v int) (Port, E.NestedError) {
	pp := Port(v)
	if err := pp.boundCheck(); err.IsNotNil() {
		return ErrPort, err
	}
	return pp, E.Nil()
}

func (p Port) boundCheck() E.NestedError {
	if p < MinPort || p > MaxPort {
		return E.Invalid("port", p)
	}
	return E.Nil()
}

const (
	MinPort  = 0
	MaxPort  = 65535
	ErrPort  = Port(-1)
	NoPort   = Port(-1)
	ZeroPort = Port(0)
)
