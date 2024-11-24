package types

import (
	"strconv"
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

var validPorts = []string{
	"1234:5678",
	"0:2345",
	"2345",
}

var invalidPorts = []string{
	"",
	"123:",
	"0:",
	":1234",
	"qwerty",
	"asdfgh:asdfgh",
	"1234:asdfgh",
}

var outOfRangePorts = []string{
	"-1:1234",
	"1234:-1",
	"65536",
	"0:65536",
}

var tooManyColonsPorts = []string{
	"1234:1234:1234",
}

func TestStreamPort(t *testing.T) {
	for _, port := range validPorts {
		_, err := ValidateStreamPort(port)
		ExpectNoError(t, err)
	}
	for _, port := range invalidPorts {
		_, err := ValidateStreamPort(port)
		ExpectError2(t, port, strconv.ErrSyntax, err)
	}
	for _, port := range outOfRangePorts {
		_, err := ValidateStreamPort(port)
		ExpectError2(t, port, ErrPortOutOfRange, err)
	}
	for _, port := range tooManyColonsPorts {
		_, err := ValidateStreamPort(port)
		ExpectError2(t, port, ErrStreamPortTooManyColons, err)
	}
}
