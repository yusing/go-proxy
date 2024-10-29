package middleware

import (
	"net"
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/types"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type cidrWhitelist struct {
	*cidrWhitelistOpts
	m *Middleware
}

type cidrWhitelistOpts struct {
	Allow      []*types.CIDR `json:"allow"`
	StatusCode int           `json:"statusCode"`
	Message    string        `json:"message"`

	cachedAddr F.Map[string, bool] // cache for trusted IPs
}

var CIDRWhiteList = &cidrWhitelist{
	m: &Middleware{withOptions: NewCIDRWhitelist},
}

var cidrWhitelistDefaults = func() *cidrWhitelistOpts {
	return &cidrWhitelistOpts{
		Allow:      []*types.CIDR{},
		StatusCode: http.StatusForbidden,
		Message:    "IP not allowed",
		cachedAddr: F.NewMapOf[string, bool](),
	}
}

func NewCIDRWhitelist(opts OptionsRaw) (*Middleware, E.Error) {
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
		return nil, E.New("no allowed CIDRs")
	}
	return wl.m, nil
}

func (wl *cidrWhitelist) checkIP(next http.HandlerFunc, w ResponseWriter, r *Request) {
	var allow, ok bool
	if allow, ok = wl.cachedAddr.Load(r.RemoteAddr); !ok {
		ipStr, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ipStr = r.RemoteAddr
		}
		ip := net.ParseIP(ipStr)
		for _, cidr := range wl.cidrWhitelistOpts.Allow {
			if cidr.Contains(ip) {
				wl.cachedAddr.Store(r.RemoteAddr, true)
				allow = true
				wl.m.AddTracef("client %s is allowed", ipStr).With("allowed CIDR", cidr)
				break
			}
		}
		if !allow {
			wl.cachedAddr.Store(r.RemoteAddr, false)
			wl.m.AddTracef("client %s is forbidden", ipStr).With("allowed CIDRs", wl.cidrWhitelistOpts.Allow)
		}
	}
	if !allow {
		w.WriteHeader(wl.StatusCode)
		w.Write([]byte(wl.Message))
		return
	}

	next(w, r)
}
