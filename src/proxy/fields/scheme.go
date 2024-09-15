package fields

import (
	"strings"

	E "github.com/yusing/go-proxy/error"
)

type Scheme string

func NewScheme(s string) (Scheme, E.NestedError) {
	switch s {
	case "http", "https", "tcp", "udp":
		return Scheme(s), E.Nil()
	}
	return "", E.Invalid("scheme", s)
}

func NewSchemeFromPort(p string) (Scheme, E.NestedError) {
	var s string
	switch {
	case strings.ContainsRune(p, ':'):
		s = "tcp"
	case strings.HasSuffix(p, "443"):
		s = "https"
	default:
		s = "http"
	}
	return Scheme(s), E.Nil()
}

func (s Scheme) IsHTTP() bool   { return s == "http" }
func (s Scheme) IsHTTPS() bool  { return s == "https" }
func (s Scheme) IsTCP() bool    { return s == "tcp" }
func (s Scheme) IsUDP() bool    { return s == "udp" }
func (s Scheme) IsStream() bool { return s.IsTCP() || s.IsUDP() }
