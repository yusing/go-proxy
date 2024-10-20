package fields

import (
	"strconv"

	E "github.com/yusing/go-proxy/internal/error"
)

type Port int

func ValidatePort[String ~string](v String) (Port, E.Error) {
	p, err := strconv.Atoi(string(v))
	if err != nil {
		return ErrPort, E.Invalid("port number", v).With(err)
	}
	return ValidatePortInt(p)
}

func ValidatePortInt[Int int | uint16](v Int) (Port, E.Error) {
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
	MinPort = 0
	MaxPort = 65535
	ErrPort = Port(-1)
	NoPort  = Port(0)
)
