package fields

import (
	"testing"

	E "github.com/yusing/go-proxy/error"
	U "github.com/yusing/go-proxy/utils/testing"
)

var validPorts = []string{
	"1234:5678",
	"0:2345",
	"2345",
	"1234:postgres",
}

var invalidPorts = []string{
	"",
	"123:",
	"0:",
	":1234",
	"1234:1234:1234",
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

func TestStreamPort(t *testing.T) {
	for _, port := range validPorts {
		_, err := ValidateStreamPort(port)
		U.ExpectNoError(t, err.Error())
	}
	for _, port := range invalidPorts {
		_, err := ValidateStreamPort(port)
		U.ExpectError2(t, port, E.ErrInvalid, err.Error())
	}
	for _, port := range outOfRangePorts {
		_, err := ValidateStreamPort(port)
		U.ExpectError2(t, port, E.ErrOutOfRange, err.Error())
	}
}
