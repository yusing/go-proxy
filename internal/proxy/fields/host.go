package fields

import (
	E "github.com/yusing/go-proxy/internal/error"
)

type (
	Host      string
	Subdomain = Alias
)

func ValidateHost[String ~string](s String) (Host, E.Error) {
	return Host(s), nil
}
