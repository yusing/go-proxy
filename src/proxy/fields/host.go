package fields

import (
	E "github.com/yusing/go-proxy/error"
)

type Host string
type Subdomain = Alias

func NewHost(s string) (Host, E.NestedError) {
	return Host(s), E.Nil()
}
