package middleware

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

const (
	cfIPv4CIDRsEndpoint        = "https://www.cloudflare.com/ips-v4"
	cfIPv6CIDRsEndpoint        = "https://www.cloudflare.com/ips-v6"
	cfCIDRsUpdateInterval      = time.Hour
	cfCIDRsUpdateRetryInterval = 3 * time.Second
)

var (
	cfCIDRsLastUpdate time.Time
	cfCIDRsMu         sync.Mutex
	cfCIDRsLogger     = logger.With().Str("name", "CloudflareRealIP").Logger()
)

var CloudflareRealIP = &realIP{
	m: &Middleware{withOptions: NewCloudflareRealIP},
}

func NewCloudflareRealIP(_ OptionsRaw) (*Middleware, E.Error) {
	cri := new(realIP)
	cri.m = &Middleware{
		impl: cri,
		before: func(next http.HandlerFunc, w ResponseWriter, r *Request) {
			cidrs := tryFetchCFCIDR()
			if cidrs != nil {
				cri.From = cidrs
			}
			cri.setRealIP(r)
			next(w, r)
		},
	}
	cri.realIPOpts = &realIPOpts{
		Header:    "CF-Connecting-IP",
		Recursive: true,
	}
	return cri.m, nil
}

func tryFetchCFCIDR() (cfCIDRs []*types.CIDR) {
	if time.Since(cfCIDRsLastUpdate) < cfCIDRsUpdateInterval {
		return
	}

	cfCIDRsMu.Lock()
	defer cfCIDRsMu.Unlock()

	if time.Since(cfCIDRsLastUpdate) < cfCIDRsUpdateInterval {
		return
	}

	if common.IsTest {
		cfCIDRs = []*types.CIDR{
			{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 0, 0, 0)},
			{IP: net.IPv4(10, 0, 0, 0), Mask: net.IPv4Mask(255, 0, 0, 0)},
			{IP: net.IPv4(172, 16, 0, 0), Mask: net.IPv4Mask(255, 255, 0, 0)},
			{IP: net.IPv4(192, 168, 0, 0), Mask: net.IPv4Mask(255, 255, 255, 0)},
		}
	} else {
		cfCIDRs = make([]*types.CIDR, 0, 30)
		err := errors.Join(
			fetchUpdateCFIPRange(cfIPv4CIDRsEndpoint, cfCIDRs),
			fetchUpdateCFIPRange(cfIPv6CIDRsEndpoint, cfCIDRs),
		)
		if err != nil {
			cfCIDRsLastUpdate = time.Now().Add(-cfCIDRsUpdateRetryInterval - cfCIDRsUpdateInterval)
			cfCIDRsLogger.Err(err).Msg("failed to update cloudflare range, retry in " + strutils.FormatDuration(cfCIDRsUpdateRetryInterval))
			return nil
		}
	}

	cfCIDRsLastUpdate = time.Now()
	cfCIDRsLogger.Info().Msg("cloudflare CIDR range updated")
	return
}

func fetchUpdateCFIPRange(endpoint string, cfCIDRs []*types.CIDR) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(body), "\n") {
		if line == "" {
			continue
		}
		_, cidr, err := net.ParseCIDR(line)
		if err != nil {
			return fmt.Errorf("cloudflare responeded an invalid CIDR: %s", line)
		} else {
			cfCIDRs = append(cfCIDRs, (*types.CIDR)(cidr))
		}
	}

	return nil
}
