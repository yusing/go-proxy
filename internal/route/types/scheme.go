package types

import (
	E "github.com/yusing/go-proxy/internal/error"
)

type Scheme string

var ErrInvalidScheme = E.New("invalid scheme")

const (
	SchemeHTTP       Scheme = "http"
	SchemeHTTPS      Scheme = "https"
	SchemeTCP        Scheme = "tcp"
	SchemeUDP        Scheme = "udp"
	SchemeFileServer Scheme = "fileserver"
)

func (s Scheme) Validate() E.Error {
	switch s {
	case SchemeHTTP, SchemeHTTPS,
		SchemeTCP, SchemeUDP, SchemeFileServer:
		return nil
	}
	return ErrInvalidScheme.Subject(string(s))
}

func (s Scheme) IsReverseProxy() bool { return s == SchemeHTTP || s == SchemeHTTPS }
func (s Scheme) IsStream() bool       { return s == SchemeTCP || s == SchemeUDP }
