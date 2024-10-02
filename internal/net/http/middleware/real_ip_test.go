package middleware

import (
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/yusing/go-proxy/internal/types"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestSetRealIPOpts(t *testing.T) {
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

	ri, err := NewRealIP(opts)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, ri.impl.(*realIP).Header, optExpected.Header)
	ExpectEqual(t, ri.impl.(*realIP).Recursive, optExpected.Recursive)
	for i, CIDR := range ri.impl.(*realIP).From {
		ExpectEqual(t, CIDR.String(), optExpected.From[i].String())
	}
}

func TestSetRealIP(t *testing.T) {
	const (
		testHeader = "X-Real-IP"
		testRealIP = "192.168.1.1"
	)
	opts := OptionsRaw{
		"header": testHeader,
		"from":   []string{"0.0.0.0/0"},
	}
	optsMr := OptionsRaw{
		"set_headers": map[string]string{testHeader: testRealIP},
	}
	realip, err := NewRealIP(opts)
	ExpectNoError(t, err.Error())

	mr, err := NewModifyRequest(optsMr)
	ExpectNoError(t, err.Error())

	mid := BuildMiddlewareFromChain("test", []*Middleware{mr, realip})

	result, err := newMiddlewareTest(mid, nil)
	ExpectNoError(t, err.Error())
	t.Log(traces)
	ExpectEqual(t, result.ResponseStatus, http.StatusOK)
	ExpectEqual(t, strings.Split(result.RemoteAddr, ":")[0], testRealIP)
	ExpectEqual(t, result.RequestHeaders.Get(xForwardedFor), testRealIP)
}
