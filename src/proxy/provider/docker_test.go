package provider

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/yusing/go-proxy/common"
	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	P "github.com/yusing/go-proxy/proxy"
	T "github.com/yusing/go-proxy/proxy/fields"

	. "github.com/yusing/go-proxy/utils/testing"
)

var dummyNames = []string{"/a"}

func TestApplyLabelFieldValidity(t *testing.T) {
	pathPatterns := `
- /
- POST /upload/{$}
- GET /static
`[1:]
	pathPatternsExpect := []string{
		"/",
		"POST /upload/{$}",
		"GET /static",
	}
	middlewaresExpect := D.NestedLabelMap{
		"middleware1": {
			"prop1": "value1",
			"prop2": "value2",
		},
		"middleware2": {
			"prop3": "value3",
			"prop4": "value4",
		},
	}
	var p DockerProvider
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:                          "a,b",
			D.LabelIdleTimeout:                      common.IdleTimeoutDefault,
			D.LabelStopMethod:                       common.StopMethodDefault,
			D.LabelStopSignal:                       "SIGTERM",
			D.LabelStopTimeout:                      common.StopTimeoutDefault,
			D.LabelWakeTimeout:                      common.WakeTimeoutDefault,
			"proxy.*.no_tls_verify":                 "true",
			"proxy.*.scheme":                        "https",
			"proxy.*.host":                          "app",
			"proxy.*.port":                          "4567",
			"proxy.a.no_tls_verify":                 "true",
			"proxy.a.path_patterns":                 pathPatterns,
			"proxy.a.middlewares.middleware1.prop1": "value1",
			"proxy.a.middlewares.middleware1.prop2": "value2",
			"proxy.a.middlewares.middleware2.prop3": "value3",
			"proxy.a.middlewares.middleware2.prop4": "value4",
		},
		Ports: []types.Port{
			{Type: "tcp", PrivatePort: 4567, PublicPort: 8888},
		}}, ""))
	ExpectNoError(t, err.Error())

	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)

	ExpectEqual(t, a.Scheme, "https")
	ExpectEqual(t, b.Scheme, "https")

	ExpectEqual(t, a.Host, "app")
	ExpectEqual(t, b.Host, "app")

	ExpectEqual(t, a.Port, "8888")
	ExpectEqual(t, b.Port, "8888")

	ExpectTrue(t, a.NoTLSVerify)
	ExpectTrue(t, b.NoTLSVerify)

	ExpectDeepEqual(t, a.PathPatterns, pathPatternsExpect)
	ExpectEqual(t, len(b.PathPatterns), 0)

	ExpectDeepEqual(t, a.Middlewares, middlewaresExpect)
	ExpectEqual(t, len(b.Middlewares), 0)

	ExpectEqual(t, a.IdleTimeout, common.IdleTimeoutDefault)
	ExpectEqual(t, b.IdleTimeout, common.IdleTimeoutDefault)

	ExpectEqual(t, a.StopTimeout, common.StopTimeoutDefault)
	ExpectEqual(t, b.StopTimeout, common.StopTimeoutDefault)

	ExpectEqual(t, a.StopMethod, common.StopMethodDefault)
	ExpectEqual(t, b.StopMethod, common.StopMethodDefault)

	ExpectEqual(t, a.WakeTimeout, common.WakeTimeoutDefault)
	ExpectEqual(t, b.WakeTimeout, common.WakeTimeoutDefault)

	ExpectEqual(t, a.StopSignal, "SIGTERM")
	ExpectEqual(t, b.StopSignal, "SIGTERM")
}

func TestApplyLabel(t *testing.T) {
	var p DockerProvider
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:          "a,b,c",
			"proxy.a.no_tls_verify": "true",
			"proxy.a.port":          "3333",
			"proxy.b.port":          "1234",
			"proxy.c.scheme":        "https",
		},
		Ports: []types.Port{
			{Type: "tcp", PrivatePort: 3333, PublicPort: 1111},
			{Type: "tcp", PrivatePort: 4444, PublicPort: 1234},
		}}, "",
	))
	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)
	c, ok := entries.Load("c")
	ExpectTrue(t, ok)

	ExpectNoError(t, err.Error())
	ExpectEqual(t, a.Scheme, "http")
	ExpectEqual(t, a.Port, "1111")
	ExpectEqual(t, a.NoTLSVerify, true)
	ExpectEqual(t, b.Scheme, "http")
	ExpectEqual(t, b.Port, "1234")
	ExpectEqual(t, c.Scheme, "https")
	// map does not necessary follow the order above
	ExpectEqualAny(t, c.Port, []string{"1111", "1234"})
}

func TestApplyLabelWithRef(t *testing.T) {
	var p DockerProvider
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:    "a,b,c",
			"proxy.#1.host":   "localhost",
			"proxy.#1.port":   "4444",
			"proxy.#2.port":   "9999",
			"proxy.#3.port":   "1111",
			"proxy.#3.scheme": "https",
		},
		Ports: []types.Port{
			{Type: "tcp", PrivatePort: 3333, PublicPort: 9999},
			{Type: "tcp", PrivatePort: 4444, PublicPort: 5555},
			{Type: "tcp", PrivatePort: 1111, PublicPort: 2222},
		}}, ""))
	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)
	c, ok := entries.Load("c")
	ExpectTrue(t, ok)

	ExpectNoError(t, err.Error())
	ExpectEqual(t, a.Scheme, "http")
	ExpectEqual(t, a.Host, "localhost")
	ExpectEqual(t, a.Port, "5555")
	ExpectEqual(t, b.Port, "9999")
	ExpectEqual(t, c.Scheme, "https")
	ExpectEqual(t, c.Port, "2222")
}

func TestApplyLabelWithRefIndexError(t *testing.T) {
	var p DockerProvider
	var c = D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:    "a,b",
			"proxy.#1.host":   "localhost",
			"proxy.#4.scheme": "https",
		}}, "")
	_, err := p.entriesFromContainerLabels(c)
	ExpectError(t, E.ErrOutOfRange, err.Error())
	ExpectTrue(t, strings.Contains(err.String(), "index out of range"))

	_, err = p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:  "a,b",
			"proxy.#0.host": "localhost",
		}}, ""))
	ExpectError(t, E.ErrOutOfRange, err.Error())
	ExpectTrue(t, strings.Contains(err.String(), "index out of range"))
}

func TestStreamDefaultValues(t *testing.T) {
	var p DockerProvider
	var c = D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:          "a",
			"proxy.*.no_tls_verify": "true",
		},
		Ports: []types.Port{
			{Type: "udp", PrivatePort: 1234, PublicPort: 5678},
		}}, "",
	)
	entries, err := p.entriesFromContainerLabels(c)
	ExpectNoError(t, err.Error())

	raw, ok := entries.Load("a")
	ExpectTrue(t, ok)

	entry, err := P.ValidateEntry(raw)
	ExpectNoError(t, err.Error())

	a := ExpectType[*P.StreamEntry](t, entry)
	ExpectEqual(t, a.Scheme.ListeningScheme, T.Scheme("udp"))
	ExpectEqual(t, a.Scheme.ProxyScheme, T.Scheme("udp"))
	ExpectEqual(t, a.Port.ListeningPort, 0)
	ExpectEqual(t, a.Port.ProxyPort, 5678)
}

func TestExplicitExclude(t *testing.T) {
	var p DockerProvider
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:          "a",
			D.LabelExclude:          "true",
			"proxy.a.no_tls_verify": "true",
		}}, ""))
	ExpectNoError(t, err.Error())

	_, ok := entries.Load("a")
	ExpectFalse(t, ok)
}

func TestImplicitExclude(t *testing.T) {
	var p DockerProvider
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:          "a",
			"proxy.a.no_tls_verify": "true",
		},
		State: "running",
	}, ""))
	ExpectNoError(t, err.Error())

	_, ok := entries.Load("a")
	ExpectFalse(t, ok)
}

func TestImplicitExcludeNoExposedPort(t *testing.T) {
	var p DockerProvider
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Image: "redis",
		Names: []string{"redis"},
		Ports: []types.Port{
			{Type: "tcp", PrivatePort: 6379, PublicPort: 0}, // not exposed
		},
		State: "running",
	}, ""))
	ExpectNoError(t, err.Error())

	_, ok := entries.Load("redis")
	ExpectFalse(t, ok)
}

func TestNotExcludeSpecifiedPort(t *testing.T) {
	var p DockerProvider
	entries, err := p.entriesFromContainerLabels(D.FromDocker(&types.Container{
		Image: "redis",
		Names: []string{"redis"},
		Ports: []types.Port{
			{Type: "tcp", PrivatePort: 6379, PublicPort: 0}, // not exposed
		},
		Labels: map[string]string{
			"proxy.redis.port": "6379:6379", // but specified in label
		},
	}, ""))
	ExpectNoError(t, err.Error())

	_, ok := entries.Load("redis")
	ExpectTrue(t, ok)
}

func TestNotExcludeNonExposedPortHostNetwork(t *testing.T) {
	var p DockerProvider
	cont := &types.Container{
		Image: "redis",
		Names: []string{"redis"},
		Ports: []types.Port{
			{Type: "tcp", PrivatePort: 6379, PublicPort: 0}, // not exposed
		},
		Labels: map[string]string{
			"proxy.redis.port": "6379:6379",
		},
	}
	cont.HostConfig.NetworkMode = "host"

	entries, err := p.entriesFromContainerLabels(D.FromDocker(cont, ""))
	ExpectNoError(t, err.Error())

	_, ok := entries.Load("redis")
	ExpectTrue(t, ok)
}
