package fields

import (
	E "github.com/yusing/go-proxy/error"
)

type Scheme string

func NewScheme[String ~string](s String) (Scheme, E.NestedError) {
	switch s {
	case "http", "https", "tcp", "udp":
		return Scheme(s), nil
	}
	return "", E.Invalid("scheme", s)
}

func (s Scheme) IsHTTP() bool   { return s == "http" }
func (s Scheme) IsHTTPS() bool  { return s == "https" }
func (s Scheme) IsTCP() bool    { return s == "tcp" }
func (s Scheme) IsUDP() bool    { return s == "udp" }
func (s Scheme) IsStream() bool { return s.IsTCP() || s.IsUDP() }
