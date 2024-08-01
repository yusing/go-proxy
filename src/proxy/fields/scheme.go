package fields

import (
	"strings"

	E "github.com/yusing/go-proxy/error"
	F "github.com/yusing/go-proxy/utils/functional"
)

type Scheme struct{ F.Stringable }

func NewScheme(s string) (*Scheme, E.NestedError) {
	switch s {
	case "http", "https", "tcp", "udp":
		return &Scheme{F.NewStringable(s)}, E.Nil()
	}
	return nil, E.Invalid("scheme", s)
}

func NewSchemeFromPort(p string) (*Scheme, E.NestedError) {
	var s string
	switch {
	case strings.ContainsRune(p, ':'):
		s = "tcp"
	case strings.HasSuffix(p, "443"):
		s = "https"
	default:
		s = "http"
	}
	return &Scheme{F.NewStringable(s)}, E.Nil()
}

func (s Scheme) IsHTTP() bool   { return s.String() == "http" }
func (s Scheme) IsHTTPS() bool  { return s.String() == "https" }
func (s Scheme) IsTCP() bool    { return s.String() == "tcp" }
func (s Scheme) IsUDP() bool    { return s.String() == "udp" }
func (s Scheme) IsStream() bool { return s.IsTCP() || s.IsUDP() }
