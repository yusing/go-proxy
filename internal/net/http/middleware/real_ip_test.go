package middleware

import (
	"net"
	"testing"

	"github.com/yusing/go-proxy/internal/types"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestSetRealIP(t *testing.T) {
	opts := OptionsRaw{
		"header": "X-Real-IP",
		"from": []string{
			"127.0.0.0/8",
			"192.168.0.0/16",
			"172.16.0.0/12",
		},
		"recursive": true,
	}
	optExpected := &realIPOpts{
		Header: "X-Real-IP",
		From: []*types.CIDR{
			{
				IP:   net.ParseIP("127.0.0.0"),
				Mask: net.IPv4Mask(255, 0, 0, 0),
			},
			{
				IP:   net.ParseIP("192.168.0.0"),
				Mask: net.IPv4Mask(255, 255, 0, 0),
			},
			{
				IP:   net.ParseIP("172.16.0.0"),
				Mask: net.IPv4Mask(255, 240, 0, 0),
			},
		},
		Recursive: true,
	}

	t.Run("set_options", func(t *testing.T) {
		ri, err := RealIP.m.WithOptionsClone(opts)
		ExpectNoError(t, err.Error())
		// ExpectEqual(t, ri.impl.(*realIP).Header, optExpected.Header)
		// ExpectDeepEqual(t, ri.impl.(*realIP).From, optExpected.From)
		// ExpectEqual(t, ri.impl.(*realIP).Recursive, optExpected.Recursive)
		ExpectDeepEqual(t, ri.impl.(*realIP).realIPOpts, optExpected)
	})

	// t.Run("request_headers", func(t *testing.T) {
	// 	result, err := newMiddlewareTest(ModifyRequest.m, &testArgs{
	// 		middlewareOpt: opts,
	// 	})
	// 	ExpectNoError(t, err.Error())
	// 	ExpectEqual(t, result.RequestHeaders.Get("User-Agent"), "go-proxy/v0.5.0")
	// 	ExpectTrue(t, slices.Contains(result.RequestHeaders.Values("Accept-Encoding"), "test-value"))
	// 	ExpectEqual(t, result.RequestHeaders.Get("Accept"), "")
	// })
}
