package types

import (
	E "github.com/yusing/go-proxy/internal/error"
)

type Scheme string

var ErrInvalidScheme = E.New("invalid scheme")

func NewScheme(s string) (Scheme, error) {
	switch s {
	case "http", "https", "tcp", "udp":
		return Scheme(s), nil
	}
	return "", ErrInvalidScheme.Subject(s)
}

func (s Scheme) IsHTTP() bool   { return s == "http" }
func (s Scheme) IsHTTPS() bool  { return s == "https" }
func (s Scheme) IsTCP() bool    { return s == "tcp" }
func (s Scheme) IsUDP() bool    { return s == "udp" }
func (s Scheme) IsStream() bool { return s.IsTCP() || s.IsUDP() }
