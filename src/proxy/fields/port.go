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
	return ValidatePortInt(p)
}

func ValidatePortInt[Int int | uint16](v Int) (Port, E.NestedError) {
	p := Port(v)
	if !p.inBound() {
		return ErrPort, E.OutOfRange("port", p)
	}
	return p, nil
}

func (p Port) inBound() bool {
	return p >= MinPort && p <= MaxPort
}

func (p Port) String() string {
	return strconv.Itoa(int(p))
}

const (
	MinPort  = 0
	MaxPort  = 65535
	ErrPort  = Port(-1)
	NoPort   = Port(-1)
	ZeroPort = Port(0)
)
