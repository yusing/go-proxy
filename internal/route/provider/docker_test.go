package provider

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/yusing/go-proxy/internal/common"
	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/route/entry"
	T "github.com/yusing/go-proxy/internal/route/types"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

var dummyNames = []string{"/a"}

const (
	testIP       = "192.168.2.100"
	testDockerIP = "172.17.0.123"
)

func makeEntries(cont *types.Container, dockerHostIP ...string) route.RawEntries {
	var p DockerProvider
	var host string
	if len(dockerHostIP) > 0 {
		host = "tcp://" + dockerHostIP[0] + ":2375"
	} else {
		host = client.DefaultDockerHost
	}
	entries := E.Must(p.entriesFromContainerLabels(D.FromDocker(cont, host)))
	entries.RangeAll(func(k string, v *route.RawEntry) {
		v.Finalize()
	})
	return entries
}

func TestExplicitOnly(t *testing.T) {
	p, err := NewDockerProvider("a!", "")
	ExpectNoError(t, err)
	ExpectTrue(t, p.IsExplicitOnly())
}

func TestApplyLabel(t *testing.T) {
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
	middlewaresExpect := map[string]map[string]any{
		"middleware1": {
			"prop1": "value1",
			"prop2": "value2",
		},
		"middleware2": {
			"prop3": "value3",
			"prop4": "value4",
		},
	}
	entries := makeEntries(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:                          "a,b",
			D.LabelIdleTimeout:                      "",
			D.LabelStopMethod:                       common.StopMethodDefault,
			D.LabelStopSignal:                       "SIGTERM",
			D.LabelStopTimeout:                      common.StopTimeoutDefault,
			D.LabelWakeTimeout:                      common.WakeTimeoutDefault,
			"proxy.*.no_tls_verify":                 "true",
			"proxy.*.scheme":                        "https",
			"proxy.*.host":                          "app",
			"proxy.*.port":                          "4567",
			"proxy.a.path_patterns":                 pathPatterns,
			"proxy.a.middlewares.middleware1.prop1": "value1",
			"proxy.a.middlewares.middleware1.prop2": "value2",
			"proxy.a.middlewares.middleware2.prop3": "value3",
			"proxy.a.middlewares.middleware2.prop4": "value4",
			"proxy.a.homepage.show":                 "true",
			"proxy.a.homepage.icon":                 "png/example.png",
			"proxy.a.healthcheck.path":              "/ping",
			"proxy.a.healthcheck.interval":          "10s",
		},
	})

	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)

	ExpectEqual(t, a.Scheme, "https")
	ExpectEqual(t, b.Scheme, "https")

	ExpectEqual(t, a.Host, "app")
	ExpectEqual(t, b.Host, "app")

	ExpectEqual(t, a.Port, "4567")
	ExpectEqual(t, b.Port, "4567")

	ExpectTrue(t, a.NoTLSVerify)
	ExpectTrue(t, b.NoTLSVerify)

	ExpectDeepEqual(t, a.PathPatterns, pathPatternsExpect)
	ExpectEqual(t, len(b.PathPatterns), 0)

	ExpectDeepEqual(t, a.Middlewares, middlewaresExpect)
	ExpectEqual(t, len(b.Middlewares), 0)

	ExpectEqual(t, a.Container.IdleTimeout, "")
	ExpectEqual(t, b.Container.IdleTimeout, "")

	ExpectEqual(t, a.Container.StopTimeout, common.StopTimeoutDefault)
	ExpectEqual(t, b.Container.StopTimeout, common.StopTimeoutDefault)

	ExpectEqual(t, a.Container.StopMethod, common.StopMethodDefault)
	ExpectEqual(t, b.Container.StopMethod, common.StopMethodDefault)

	ExpectEqual(t, a.Container.WakeTimeout, common.WakeTimeoutDefault)
	ExpectEqual(t, b.Container.WakeTimeout, common.WakeTimeoutDefault)

	ExpectEqual(t, a.Container.StopSignal, "SIGTERM")
	ExpectEqual(t, b.Container.StopSignal, "SIGTERM")

	ExpectEqual(t, a.Homepage.Show, true)
	ExpectEqual(t, a.Homepage.Icon.Value, "png/example.png")
	ExpectEqual(t, a.Homepage.Icon.Extra.FileType, "png")
	ExpectEqual(t, a.Homepage.Icon.Extra.Name, "example")

	ExpectEqual(t, a.HealthCheck.Path, "/ping")
	ExpectEqual(t, a.HealthCheck.Interval, 10*time.Second)
}

func TestApplyLabelWithAlias(t *testing.T) {
	entries := makeEntries(&types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			D.LabelAliases:          "a,b,c",
			"proxy.a.no_tls_verify": "true",
			"proxy.a.port":          "3333",
			"proxy.b.port":          "1234",
			"proxy.c.scheme":        "https",
		},
	})
	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)
	c, ok := entries.Load("c")
	ExpectTrue(t, ok)

	ExpectEqual(t, a.Scheme, "http")
	ExpectEqual(t, a.Port, "3333")
	ExpectEqual(t, a.NoTLSVerify, true)
	ExpectEqual(t, b.Scheme, "http")
	ExpectEqual(t, b.Port, "1234")
	ExpectEqual(t, c.Scheme, "https")
}

func TestApplyLabelWithRef(t *testing.T) {
	entries := makeEntries(&types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			D.LabelAliases:    "a,b,c",
			"proxy.#1.host":   "localhost",
			"proxy.#1.port":   "4444",
			"proxy.#2.port":   "9999",
			"proxy.#3.port":   "1111",
			"proxy.#3.scheme": "https",
		},
	})
	a, ok := entries.Load("a")
	ExpectTrue(t, ok)
	b, ok := entries.Load("b")
	ExpectTrue(t, ok)
	c, ok := entries.Load("c")
	ExpectTrue(t, ok)

	ExpectEqual(t, a.Scheme, "http")
	ExpectEqual(t, a.Host, "localhost")
	ExpectEqual(t, a.Port, "4444")
	ExpectEqual(t, b.Port, "9999")
	ExpectEqual(t, c.Scheme, "https")
	ExpectEqual(t, c.Port, "1111")
}

func TestApplyLabelWithRefIndexError(t *testing.T) {
	c := D.FromDocker(&types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			D.LabelAliases:    "a,b",
			"proxy.#1.host":   "localhost",
			"proxy.#4.scheme": "https",
		},
	}, "")
	var p DockerProvider
	_, err := p.entriesFromContainerLabels(c)
	ExpectError(t, ErrAliasRefIndexOutOfRange, err)

	c = D.FromDocker(&types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			D.LabelAliases:  "a,b",
			"proxy.#0.host": "localhost",
		},
	}, "")
	_, err = p.entriesFromContainerLabels(c)
	ExpectError(t, ErrAliasRefIndexOutOfRange, err)
}

func TestDynamicAliases(t *testing.T) {
	c := &types.Container{
		Names: []string{"app1"},
		State: "running",
		Labels: map[string]string{
			"proxy.app1.port":         "1234",
			"proxy.app1_backend.port": "5678",
		},
	}

	entries := makeEntries(c)

	raw, ok := entries.Load("app1")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Scheme, "http")
	ExpectEqual(t, raw.Port, "1234")

	raw, ok = entries.Load("app1_backend")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Scheme, "http")
	ExpectEqual(t, raw.Port, "5678")
}

func TestDisableHealthCheck(t *testing.T) {
	c := &types.Container{
		Names: dummyNames,
		State: "running",
		Labels: map[string]string{
			"proxy.a.healthcheck.disable": "true",
			"proxy.a.port":                "1234",
		},
	}
	raw, ok := makeEntries(c).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.HealthCheck, nil)
}

func TestPublicIPLocalhost(t *testing.T) {
	c := &types.Container{Names: dummyNames, State: "running"}
	raw, ok := makeEntries(c).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Container.PublicIP, "127.0.0.1")
	ExpectEqual(t, raw.Host, raw.Container.PublicIP)
}

func TestPublicIPRemote(t *testing.T) {
	c := &types.Container{Names: dummyNames, State: "running"}
	raw, ok := makeEntries(c, testIP).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Container.PublicIP, testIP)
	ExpectEqual(t, raw.Host, raw.Container.PublicIP)
}

func TestPrivateIPLocalhost(t *testing.T) {
	c := &types.Container{
		Names: dummyNames,
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"network": {
					IPAddress: testDockerIP,
				},
			},
		},
	}
	raw, ok := makeEntries(c).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Container.PrivateIP, testDockerIP)
	ExpectEqual(t, raw.Host, raw.Container.PrivateIP)
}

func TestPrivateIPRemote(t *testing.T) {
	c := &types.Container{
		Names: dummyNames,
		State: "running",
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"network": {
					IPAddress: testDockerIP,
				},
			},
		},
	}
	raw, ok := makeEntries(c, testIP).Load("a")
	ExpectTrue(t, ok)
	ExpectEqual(t, raw.Container.PrivateIP, "")
	ExpectEqual(t, raw.Container.PublicIP, testIP)
	ExpectEqual(t, raw.Host, raw.Container.PublicIP)
}

func TestStreamDefaultValues(t *testing.T) {
	privPort := uint16(1234)
	pubPort := uint16(4567)
	privIP := "172.17.0.123"
	cont := &types.Container{
		Names: []string{"a"},
		State: "running",
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"network": {
					IPAddress: privIP,
				},
			},
		},
		Ports: []types.Port{
			{Type: "udp", PrivatePort: privPort, PublicPort: pubPort},
		},
	}

	t.Run("local", func(t *testing.T) {
		raw, ok := makeEntries(cont).Load("a")
		ExpectTrue(t, ok)
		en := E.Must(entry.ValidateEntry(raw))
		a := ExpectType[*entry.StreamEntry](t, en)
		ExpectEqual(t, a.Scheme.ListeningScheme, T.Scheme("udp"))
		ExpectEqual(t, a.Scheme.ProxyScheme, T.Scheme("udp"))
		ExpectEqual(t, a.URL.Hostname(), privIP)
		ExpectEqual(t, a.Port.ListeningPort, 0)
		ExpectEqual(t, a.Port.ProxyPort, T.Port(privPort))
	})

	t.Run("remote", func(t *testing.T) {
		raw, ok := makeEntries(cont, testIP).Load("a")
		ExpectTrue(t, ok)
		en := E.Must(entry.ValidateEntry(raw))
		a := ExpectType[*entry.StreamEntry](t, en)
		ExpectEqual(t, a.Scheme.ListeningScheme, T.Scheme("udp"))
		ExpectEqual(t, a.Scheme.ProxyScheme, T.Scheme("udp"))
		ExpectEqual(t, a.URL.Hostname(), testIP)
		ExpectEqual(t, a.Port.ListeningPort, 0)
		ExpectEqual(t, a.Port.ProxyPort, T.Port(pubPort))
	})
}

func TestExplicitExclude(t *testing.T) {
	_, ok := makeEntries(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LabelAliases:          "a",
			D.LabelExclude:          "true",
			"proxy.a.no_tls_verify": "true",
		},
	}, "").Load("a")
	ExpectFalse(t, ok)
}

func TestImplicitExcludeDatabase(t *testing.T) {
	t.Run("mount path detection", func(t *testing.T) {
		_, ok := makeEntries(&types.Container{
			Names: dummyNames,
			Mounts: []types.MountPoint{
				{Source: "/data", Destination: "/var/lib/postgresql/data"},
			},
		}).Load("a")
		ExpectFalse(t, ok)
	})
	t.Run("exposed port detection", func(t *testing.T) {
		_, ok := makeEntries(&types.Container{
			Names: dummyNames,
			Ports: []types.Port{
				{Type: "tcp", PrivatePort: 5432, PublicPort: 5432},
			},
		}).Load("a")
		ExpectFalse(t, ok)
	})
}
