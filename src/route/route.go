package route

import (
	"fmt"
	"net/url"

	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	P "github.com/yusing/go-proxy/proxy"
	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	Route interface {
		RouteImpl
		Entry() *M.RawEntry
		Type() RouteType
		URL() *url.URL
	}
	Routes    = F.Map[string, Route]
	RouteType string

	RouteImpl interface {
		Start() E.NestedError
		Stop() E.NestedError
		String() string
	}
	route struct {
		RouteImpl
		type_ RouteType
		entry *M.RawEntry
	}
)

const (
	RouteTypeStream       RouteType = "stream"
	RouteTypeReverseProxy RouteType = "reverse_proxy"
)

// function alias
var NewRoutes = F.NewMapOf[string, Route]

func NewRoute(en *M.RawEntry) (Route, E.NestedError) {
	rt, err := P.ValidateEntry(en)
	if err != nil {
		return nil, err
	}

	var t RouteType

	switch e := rt.(type) {
	case *P.StreamEntry:
		rt, err = NewStreamRoute(e)
		t = RouteTypeStream
	case *P.ReverseProxyEntry:
		rt, err = NewHTTPRoute(e)
		t = RouteTypeReverseProxy
	default:
		panic("bug: should not reach here")
	}
	return &route{RouteImpl: rt.(RouteImpl), entry: en, type_: t}, err
}

func (rt *route) Entry() *M.RawEntry {
	return rt.entry
}

func (rt *route) Type() RouteType {
	return rt.type_
}

func (rt *route) URL() *url.URL {
	url, _ := url.Parse(fmt.Sprintf("%s://%s", rt.entry.Scheme, rt.entry.Host))
	return url
}

func FromEntries(entries M.RawEntries) (Routes, E.NestedError) {
	b := E.NewBuilder("errors in routes")

	routes := NewRoutes()
	entries.RangeAll(func(alias string, entry *M.RawEntry) {
		entry.Alias = alias
		r, err := NewRoute(entry)
		if err.HasError() {
			b.Add(err.Subject(alias))
		} else {
			routes.Store(alias, r)
		}
	})

	return routes, b.Build()
}
