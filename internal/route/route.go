package route

import (
	"github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	url "github.com/yusing/go-proxy/internal/net/types"
	P "github.com/yusing/go-proxy/internal/proxy"
	"github.com/yusing/go-proxy/internal/types"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	RouteType string
	Route     struct {
		_ U.NoCopy
		impl
		Type  RouteType
		Entry *types.RawEntry
	}
	Routes = F.Map[string, *Route]

	impl interface {
		Start() E.NestedError
		Stop() E.NestedError
		Started() bool
		String() string
		URL() url.URL
	}
)

const (
	RouteTypeStream       RouteType = "stream"
	RouteTypeReverseProxy RouteType = "reverse_proxy"
)

// function alias.
var NewRoutes = F.NewMap[Routes]

func (rt *Route) Container() *docker.Container {
	if rt.Entry.Container == nil {
		return docker.DummyContainer
	}
	return rt.Entry.Container
}

func NewRoute(en *types.RawEntry) (*Route, E.NestedError) {
	entry, err := P.ValidateEntry(en)
	if err != nil {
		return nil, err
	}

	var t RouteType
	var rt impl

	switch e := entry.(type) {
	case *P.StreamEntry:
		t = RouteTypeStream
		rt, err = NewStreamRoute(e)
	case *P.ReverseProxyEntry:
		t = RouteTypeReverseProxy
		rt, err = NewHTTPRoute(e)
	default:
		panic("bug: should not reach here")
	}
	if err != nil {
		return nil, err
	}
	return &Route{
		impl:  rt,
		Type:  t,
		Entry: en,
	}, nil
}

func FromEntries(entries types.RawEntries) (Routes, E.NestedError) {
	b := E.NewBuilder("errors in routes")

	routes := NewRoutes()
	entries.RangeAll(func(alias string, entry *types.RawEntry) {
		entry.Alias = alias
		r, err := NewRoute(entry)
		if err != nil {
			b.Add(err.Subject(alias))
		} else {
			routes.Store(alias, r)
		}
	})

	return routes, b.Build()
}
