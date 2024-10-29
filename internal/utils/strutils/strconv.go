package strutils

import (
	"errors"
	"strconv"

	E "github.com/yusing/go-proxy/internal/error"
)

func Atoi(s string) (int, E.Error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return val, E.From(errors.Unwrap(err)).Subject(s)
	}

	return val, nil
}
