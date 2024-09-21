package provider

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/yusing/go-proxy/common"
	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	F "github.com/yusing/go-proxy/utils/functional"
	. "github.com/yusing/go-proxy/utils/testing"
)

func get[KT comparable, VT any](m F.Map[KT, VT], key KT) VT {
	v, _ := m.Load(key)
	return v
}

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
	setHeaders := `
X_Custom_Header1: value1
X_Custom_Header1: value2
X_Custom_Header2: value3
`[1:]
	setHeadersExpect := map[string]string{
		"X_Custom_Header1": "value1, value2",
		"X_Custom_Header2": "value3",
	}
	hideHeaders := `
- X-Custom-Header1
- X-Custom-Header2
`[1:]
	hideHeadersExpect := []string{
		"X-Custom-Header1",
		"X-Custom-Header2",
	}
	var p DockerProvider
	var c = D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LableAliases:          "a,b",
			D.LabelIdleTimeout:      common.IdleTimeoutDefault,
			D.LabelStopMethod:       common.StopMethodDefault,
			D.LabelStopSignal:       "SIGTERM",
			D.LabelStopTimeout:      common.StopTimeoutDefault,
			D.LabelWakeTimeout:      common.WakeTimeoutDefault,
			"proxy.*.no_tls_verify": "true",
			"proxy.*.scheme":        "https",
			"proxy.*.host":          "app",
			"proxy.*.port":          "4567",
			"proxy.a.no_tls_verify": "true",
			"proxy.a.path_patterns": pathPatterns,
			"proxy.a.set_headers":   setHeaders,
			"proxy.a.hide_headers":  hideHeaders,
		}}, "")
	entries, err := p.entriesFromContainerLabels(c)
	ExpectNoError(t, err.Error())
	a := get(entries, "a")
	b := get(entries, "b")

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

	ExpectDeepEqual(t, a.SetHeaders, setHeadersExpect)
	ExpectEqual(t, len(b.SetHeaders), 0)

	ExpectDeepEqual(t, a.HideHeaders, hideHeadersExpect)
	ExpectEqual(t, len(b.HideHeaders), 0)

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
	var c = D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LableAliases:          "a,b,c",
			"proxy.a.no_tls_verify": "true",
			"proxy.b.port":          "1234",
			"proxy.c.scheme":        "https",
		}}, "")
	entries, err := p.entriesFromContainerLabels(c)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, get(entries, "a").NoTLSVerify, true)
	ExpectEqual(t, get(entries, "b").Port, "1234")
	ExpectEqual(t, get(entries, "c").Scheme, "https")
}

func TestApplyLabelWithRef(t *testing.T) {
	var p DockerProvider
	var c = D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LableAliases:    "a,b,c",
			"proxy.$1.host":   "localhost",
			"proxy.$2.port":   "1234",
			"proxy.$3.scheme": "https",
		}}, "")
	entries, err := p.entriesFromContainerLabels(c)
	ExpectNoError(t, err.Error())
	ExpectEqual(t, get(entries, "a").Host, "localhost")
	ExpectEqual(t, get(entries, "b").Port, "1234")
	ExpectEqual(t, get(entries, "c").Scheme, "https")
}

func TestApplyLabelWithRefIndexError(t *testing.T) {
	var p DockerProvider
	var c = D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LableAliases:    "a,b",
			"proxy.$1.host":   "localhost",
			"proxy.$4.scheme": "https",
		}}, "")
	_, err := p.entriesFromContainerLabels(c)
	ExpectError(t, E.ErrInvalid, err.Error())
	ExpectTrue(t, strings.Contains(err.String(), "index out of range"))

	c = D.FromDocker(&types.Container{
		Names: dummyNames,
		Labels: map[string]string{
			D.LableAliases:  "a,b",
			"proxy.$0.host": "localhost",
		}}, "")
	_, err = p.entriesFromContainerLabels(c)
	ExpectError(t, E.ErrInvalid, err.Error())
	ExpectTrue(t, strings.Contains(err.String(), "index out of range"))
}
