package middleware

import (
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
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
	From []*net.IPNet
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
			"from":      CIDRListParser,
			"recursive": D.BoolParser,
		},
		withOptions: NewRealIP,
	},
}

var realIPOptsDefault = func() *realIPOpts {
	return &realIPOpts{
		Header: "X-Real-IP",
		From: []*net.IPNet{
			{IP: net.IPv4(127, 0, 0, 1), Mask: net.CIDRMask(8, 32)},
			{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
			{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
			{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
			{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)},
			{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)},
		},
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

func CIDRListParser(s string) (any, E.NestedError) {
	sl, err := D.YamlStringListParser(s)
	if err != nil {
		return nil, err
	}

	b := E.NewBuilder("invalid CIDR(s)")

	CIDRs := sl.([]string)
	res := make([]*net.IPNet, 0, len(CIDRs))

	for _, cidr := range CIDRs {
		if !strings.Contains(cidr, "/") {
			cidr += "/32" // single IP
		}
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			b.Add(E.Invalid("CIDR", cidr))
			continue
		}
		res = append(res, ipnet)
	}
	return res, b.Build()
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
		realIPLogger.Debugf("failed to split host port from %s: %s", req.RemoteAddr, err)
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

	realIPLogger.Debugf("real ip %s", lastNonTrustedIP)
}
