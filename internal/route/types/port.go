package types

import (
	"strconv"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type Port int

var ErrPortOutOfRange = E.New("port out of range")

func ValidatePort[String ~string](v String) (Port, error) {
	p, err := strutils.Atoi(string(v))
	if err != nil {
		return ErrPort, err
	}
	return ValidatePortInt(p)
}

func ValidatePortInt[Int int | uint16](v Int) (Port, error) {
	p := Port(v)
	if !p.inBound() {
		return ErrPort, ErrPortOutOfRange.Subject(strconv.Itoa(int(p)))
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
