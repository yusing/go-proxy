package fields

import (
	"strconv"

	E "github.com/yusing/go-proxy/error"
)

type Port int

func ValidatePort(v string) (Port, E.NestedError) {
	p, err := strconv.Atoi(v)
	if err != nil {
		return ErrPort, E.Invalid("port number", v).With(err)
	}
	return NewPortInt(p)
}

func NewPortInt[Int int | uint16](v Int) (Port, E.NestedError) {
	pp := Port(v)
	if err := pp.boundCheck(); err.HasError() {
		return ErrPort, err
	}
	return pp, nil
}

func (p Port) boundCheck() E.NestedError {
	if p < MinPort || p > MaxPort {
		return E.Invalid("port", p)
	}
	return nil
}

const (
	MinPort  = 0
	MaxPort  = 65535
	ErrPort  = Port(-1)
	NoPort   = Port(-1)
	ZeroPort = Port(0)
)
