package route

import (
	"fmt"
	"net/url"

	E "github.com/yusing/go-proxy/internal/error"
	P "github.com/yusing/go-proxy/internal/proxy"
	"github.com/yusing/go-proxy/internal/types"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	Route interface {
		RouteImpl
		Entry() *types.RawEntry
		Type() RouteType
		URL() *url.URL
	}
	Routes = F.Map[string, Route]

	RouteImpl interface {
		Start() E.NestedError
		Stop() E.NestedError
		Started() bool
		String() string
	}
	RouteType string
	route     struct {
		RouteImpl
		type_ RouteType
		entry *types.RawEntry
	}
)

const (
	RouteTypeStream       RouteType = "stream"
	RouteTypeReverseProxy RouteType = "reverse_proxy"
)

// function alias
var NewRoutes = F.NewMapOf[string, Route]

func NewRoute(en *types.RawEntry) (Route, E.NestedError) {
	entry, err := P.ValidateEntry(en)
	if err != nil {
		return nil, err
	}

	var t RouteType
	var rt RouteImpl
	switch e := entry.(type) {
	case *P.StreamEntry:
		rt, err = NewStreamRoute(e)
		t = RouteTypeStream
	case *P.ReverseProxyEntry:
		rt, err = NewHTTPRoute(e)
		t = RouteTypeReverseProxy
	default:
		panic("bug: should not reach here")
	}
	if err != nil {
		return nil, err
	}
	return &route{RouteImpl: rt, entry: en, type_: t}, nil
}

func (rt *route) Entry() *types.RawEntry {
	return rt.entry
}

func (rt *route) Type() RouteType {
	return rt.type_
}

func (rt *route) URL() *url.URL {
	url, _ := url.Parse(fmt.Sprintf("%s://%s:%s", rt.entry.Scheme, rt.entry.Host, rt.entry.Port))
	return url
}

func FromEntries(entries types.RawEntries) (Routes, E.NestedError) {
	b := E.NewBuilder("errors in routes")

	routes := NewRoutes()
	entries.RangeAll(func(alias string, entry *types.RawEntry) {
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
