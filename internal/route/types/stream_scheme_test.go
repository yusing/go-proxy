package types

import (
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

var (
	validStreamSchemes = []string{
		"tcp:tcp",
		"tcp:udp",
		"udp:tcp",
		"udp:udp",
		"tcp",
		"udp",
	}

	invalidStreamSchemes = []string{
		"tcp:tcp:",
		"tcp:",
		":udp:",
		":udp",
		"top",
	}
)

func TestNewStreamScheme(t *testing.T) {
	for _, s := range validStreamSchemes {
		_, err := ValidateStreamScheme(s)
		ExpectNoError(t, err)
	}
	for _, s := range invalidStreamSchemes {
		_, err := ValidateStreamScheme(s)
		ExpectError(t, ErrInvalidScheme, err)
	}
}
