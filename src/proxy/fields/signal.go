package fields

import (
	E "github.com/yusing/go-proxy/error"
)

type Signal string

func ValidateSignal(s string) (Signal, E.NestedError) {
	switch s {
	case "", "SIGINT", "SIGTERM", "SIGHUP", "SIGQUIT",
		"INT", "TERM", "HUP", "QUIT":
		return Signal(s), nil
	}

	return "", E.Invalid("signal", s)
}
