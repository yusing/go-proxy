package route

import (
	"strings"

	"github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	url "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/route/entry"
	"github.com/yusing/go-proxy/internal/route/types"
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
		Entry *RawEntry
	}
	Routes = F.Map[string, *Route]

	impl interface {
		entry.Entry
		task.TaskStarter
		task.TaskFinisher
		String() string
		TargetURL() url.URL
	}
	RawEntry   = types.RawEntry
	RawEntries = types.RawEntries
)

const (
	RouteTypeStream       RouteType = "stream"
	RouteTypeReverseProxy RouteType = "reverse_proxy"
)

// function alias.
var NewRoutes = F.NewMap[Routes]
var NewProxyEntries = types.NewProxyEntries

func (rt *Route) Container() *docker.Container {
	if rt.Entry.Container == nil {
		return docker.DummyContainer
	}
	return rt.Entry.Container
}

func NewRoute(raw *RawEntry) (*Route, E.Error) {
	raw.Finalize()
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

func FromEntries(entries RawEntries) (Routes, E.Error) {
	b := E.NewBuilder("errors in routes")

	routes := NewRoutes()
	entries.RangeAllParallel(func(alias string, en *RawEntry) {
		en.Alias = alias
		if strings.HasPrefix(alias, "x-") { // x properties
			return
		}
		r, err := NewRoute(en)
		switch {
		case err != nil:
			b.Add(err.Subject(alias))
		case entry.ShouldNotServe(r):
			return
		default:
			routes.Store(alias, r)
		}
	})

	return routes, b.Error()
}
