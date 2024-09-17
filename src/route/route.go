package route

import (
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	P "github.com/yusing/go-proxy/proxy"
	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	Route interface {
		Start() E.NestedError
		Stop() E.NestedError
		String() string
	}
	Routes = F.Map[string, Route]
)

// function alias
var NewRoutes = F.NewMap[string, Route]

func NewRoute(en *M.ProxyEntry) (Route, E.NestedError) {
	entry, err := P.NewEntry(en)
	if err.HasError() {
		return nil, err
	}
	switch e := entry.(type) {
	case *P.StreamEntry:
		return NewStreamRoute(e)
	case *P.Entry:
		return NewHTTPRoute(e)
	default:
		panic("bug: should not reach here")
	}
}
