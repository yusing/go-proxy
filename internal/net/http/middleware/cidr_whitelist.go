package middleware

import (
	"net"
	"net/http"

	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/types"
)

type cidrWhitelist struct {
	*cidrWhitelistOpts
	m *Middleware
}

type cidrWhitelistOpts struct {
	Allow      []*types.CIDR
	StatusCode int
	Message    string

	trustedAddr map[string]struct{} // cache for trusted IPs
}

var CIDRWhiteList = &cidrWhitelist{
	m: &Middleware{
		labelParserMap: D.ValueParserMap{
			"allow":      D.YamlStringListParser,
			"statusCode": D.IntParser,
		},
	},
}

var cidrWhitelistDefaults = func() *cidrWhitelistOpts {
	return &cidrWhitelistOpts{
		Allow:       []*types.CIDR{},
		StatusCode:  http.StatusForbidden,
		Message:     "IP not allowed",
		trustedAddr: make(map[string]struct{}),
	}
}

func NewCIDRWhitelist(opts OptionsRaw) (*Middleware, E.NestedError) {
	wl := new(cidrWhitelist)
	wl.m = &Middleware{
		impl:   wl,
		before: wl.checkIP,
	}
	wl.cidrWhitelistOpts = cidrWhitelistDefaults()
	err := Deserialize(opts, wl.cidrWhitelistOpts)
	if err != nil {
		return nil, err
	}
	if len(wl.cidrWhitelistOpts.Allow) == 0 {
		return nil, E.Missing("allow range")
	}
	return wl.m, nil
}

func (wl *cidrWhitelist) checkIP(next http.Handler, w ResponseWriter, r *Request) {
	var ok bool
	if _, ok = wl.trustedAddr[r.RemoteAddr]; !ok {
		ip := net.IP(r.RemoteAddr)
		for _, cidr := range wl.cidrWhitelistOpts.Allow {
			if cidr.Contains(ip) {
				wl.trustedAddr[r.RemoteAddr] = struct{}{}
				ok = true
				break
			}
		}
	}
	if !ok {
		w.WriteHeader(wl.StatusCode)
		w.Write([]byte(wl.Message))
		return
	}

	next.ServeHTTP(w, r)
}
