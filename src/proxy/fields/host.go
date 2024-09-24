package fields

import (
	E "github.com/yusing/go-proxy/error"
)

type Host string
type Subdomain = Alias

func ValidateHost[String ~string](s String) (Host, E.NestedError) {
	return Host(s), nil
}
