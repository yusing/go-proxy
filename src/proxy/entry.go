package proxy

import (
	"fmt"
	"net/http"
	"net/url"

	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	T "github.com/yusing/go-proxy/proxy/fields"
)

type (
	Entry struct { // real model after validation
		Alias        T.Alias
		Scheme       T.Scheme
		Host         T.Host
		Port         T.Port
		URL          *url.URL
		NoTLSVerify  bool
		PathPatterns T.PathPatterns
		SetHeaders   http.Header
		HideHeaders  []string
	}
	StreamEntry struct {
		Alias  T.Alias        `json:"alias"`
		Scheme T.StreamScheme `json:"scheme"`
		Host   T.Host         `json:"host"`
		Port   T.StreamPort   `json:"port"`
	}
)

func NewEntry(m *M.ProxyEntry) (any, E.NestedError) {
	m.SetDefaults()
	scheme, err := T.NewScheme(m.Scheme)
	if err.HasError() {
		return nil, err
	}
	if scheme.IsStream() {
		return validateStreamEntry(m)
	}
	return validateEntry(m, scheme)
}

func validateEntry(m *M.ProxyEntry, s T.Scheme) (*Entry, E.NestedError) {
	host, err := T.NewHost(m.Host)
	if err.HasError() {
		return nil, err
	}
	port, err := T.NewPort(m.Port)
	if err.HasError() {
		return nil, err
	}
	pathPatterns, err := T.NewPathPatterns(m.PathPatterns)
	if err.HasError() {
		return nil, err
	}
	setHeaders, err := T.NewHTTPHeaders(m.SetHeaders)
	if err.HasError() {
		return nil, err
	}
	url, err := E.Check(url.Parse(fmt.Sprintf("%s://%s:%d", s, host, port)))
	if err.HasError() {
		return nil, err
	}
	return &Entry{
		Alias:        T.NewAlias(m.Alias),
		Scheme:       s,
		Host:         host,
		Port:         port,
		URL:          url,
		NoTLSVerify:  m.NoTLSVerify,
		PathPatterns: pathPatterns,
		SetHeaders:   setHeaders,
		HideHeaders:  m.HideHeaders,
	}, E.Nil()
}

func validateStreamEntry(m *M.ProxyEntry) (*StreamEntry, E.NestedError) {
	host, err := T.NewHost(m.Host)
	if err.HasError() {
		return nil, err
	}
	port, err := T.NewStreamPort(m.Port)
	if err.HasError() {
		return nil, err
	}
	scheme, err := T.NewStreamScheme(m.Scheme)
	if err.HasError() {
		return nil, err
	}
	return &StreamEntry{
		Alias:  T.NewAlias(m.Alias),
		Scheme: *scheme,
		Host:   host,
		Port:   port,
	}, E.Nil()
}
