package middleware

import (
	"net"

	"github.com/sirupsen/logrus"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/types"
)

// https://nginx.org/en/docs/http/ngx_http_realip_module.html

type realIP struct {
	*realIPOpts
	m *Middleware
}

type realIPOpts struct {
	// Header is the name of the header to use for the real client IP
	Header string
	// From is a list of Address / CIDRs to trust
	From []*types.CIDR
	/*
		If recursive search is disabled,
		the original client address that matches one of the trusted addresses is replaced by
		the last address sent in the request header field defined by the Header field.
		If recursive search is enabled,
		the original client address that matches one of the trusted addresses is replaced by
		the last non-trusted address sent in the request header field.
	*/
	Recursive bool
}

var RealIP = &realIP{
	m: &Middleware{
		labelParserMap: D.ValueParserMap{
			"from":      D.YamlStringListParser,
			"recursive": D.BoolParser,
		},
		withOptions: NewRealIP,
	},
}

var realIPOptsDefault = func() *realIPOpts {
	return &realIPOpts{
		Header: "X-Real-IP",
		From:   []*types.CIDR{},
	}
}

var realIPLogger = logrus.WithField("middleware", "RealIP")

func NewRealIP(opts OptionsRaw) (*Middleware, E.NestedError) {
	riWithOpts := new(realIP)
	riWithOpts.m = &Middleware{
		impl:    riWithOpts,
		rewrite: riWithOpts.setRealIP,
	}
	riWithOpts.realIPOpts = realIPOptsDefault()
	err := Deserialize(opts, riWithOpts.realIPOpts)
	if err != nil {
		return nil, err
	}
	return riWithOpts.m, nil
}

func (ri *realIP) isInCIDRList(ip net.IP) bool {
	for _, CIDR := range ri.From {
		if CIDR.Contains(ip) {
			return true
		}
	}
	// not in any CIDR
	return false
}

func (ri *realIP) setRealIP(req *Request) {
	clientIPStr, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		realIPLogger.Debugf("failed to split host port %s", err)
	}
	clientIP := net.ParseIP(clientIPStr)

	var isTrusted = false
	for _, CIDR := range ri.From {
		if CIDR.Contains(clientIP) {
			isTrusted = true
			break
		}
	}
	if !isTrusted {
		realIPLogger.Debugf("client ip %s is not trusted", clientIP)
		return
	}

	var realIPs = req.Header.Values(ri.Header)
	var lastNonTrustedIP string

	if len(realIPs) == 0 {
		realIPLogger.Debugf("no real ip found in header %q", ri.Header)
		return
	}

	if !ri.Recursive {
		lastNonTrustedIP = realIPs[len(realIPs)-1]
	} else {
		for _, r := range realIPs {
			if !ri.isInCIDRList(net.ParseIP(r)) {
				lastNonTrustedIP = r
			}
		}
		if lastNonTrustedIP == "" {
			realIPLogger.Debugf("no non-trusted ip found in header %q", ri.Header)
			return
		}
	}

	req.RemoteAddr = lastNonTrustedIP
	req.Header.Set(ri.Header, lastNonTrustedIP)
	req.Header.Set("X-Real-IP", lastNonTrustedIP)
	req.Header.Set("X-Forwarded-For", lastNonTrustedIP)
}
