package route

import (
	"github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	url "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/proxy/entry"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	RouteType string
	Route     struct {
		_ U.NoCopy
		impl
		Type  RouteType
		Entry *entry.RawEntry
	}
	Routes = F.Map[string, *Route]

	impl interface {
		entry.Entry
		task.TaskStarter
		task.TaskFinisher
		String() string
		TargetURL() url.URL
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

func NewRoute(raw *entry.RawEntry) (*Route, E.Error) {
	en, err := entry.ValidateEntry(raw)
	if err != nil {
		return nil, err
	}

	var t RouteType
	var rt impl

	switch e := en.(type) {
	case *entry.StreamEntry:
		t = RouteTypeStream
		rt, err = NewStreamRoute(e)
	case *entry.ReverseProxyEntry:
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
		Entry: raw,
	}, nil
}

func FromEntries(entries entry.RawEntries) (Routes, E.Error) {
	b := E.NewBuilder("errors in routes")

	routes := NewRoutes()
	entries.RangeAllParallel(func(alias string, en *entry.RawEntry) {
		en.Alias = alias
		r, err := NewRoute(en)
		if err != nil {
			b.Add(err.Subject(alias))
		} else if entry.ShouldNotServe(r) {
			return
		} else {
			routes.Store(alias, r)
		}
	})

	return routes, b.Build()
}
