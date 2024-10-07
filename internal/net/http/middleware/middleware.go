package middleware

import (
	"encoding/json"
	"errors"
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
	Error = E.NestedError

	ReverseProxy   = gphttp.ReverseProxy
	ProxyRequest   = gphttp.ProxyRequest
	Request        = http.Request
	Response       = http.Response
	ResponseWriter = http.ResponseWriter
	Header         = http.Header
	Cookie         = http.Cookie

	BeforeFunc         func(next http.HandlerFunc, w ResponseWriter, r *Request)
	RewriteFunc        func(req *Request)
	ModifyResponseFunc func(resp *Response) error
	CloneWithOptFunc   func(opts OptionsRaw) (*Middleware, E.NestedError)

	OptionsRaw = map[string]any
	Options    any

	Middleware struct {
		name string

		before         BeforeFunc         // runs before ReverseProxy.ServeHTTP
		modifyResponse ModifyResponseFunc // runs after ReverseProxy.ModifyResponse

		withOptions CloneWithOptFunc
		impl        any

		parent   *Middleware
		children []*Middleware
		trace    bool
	}
)

var Deserialize = U.Deserialize

func Rewrite(r RewriteFunc) BeforeFunc {
	return func(next http.HandlerFunc, w ResponseWriter, req *Request) {
		r(req)
		next(w, req)
	}
}

func (m *Middleware) Name() string {
	return m.name
}

func (m *Middleware) Fullname() string {
	if m.parent != nil {
		return m.parent.Fullname() + "." + m.name
	}
	return m.name
}

func (m *Middleware) String() string {
	return m.name
}

func (m *Middleware) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(map[string]any{
		"name":    m.name,
		"options": m.impl,
	}, "", "  ")
}

func (m *Middleware) WithOptionsClone(optsRaw OptionsRaw) (*Middleware, E.NestedError) {
	if len(optsRaw) != 0 && m.withOptions != nil {
		if mWithOpt, err := m.withOptions(optsRaw); err != nil {
			return nil, err
		} else {
			return mWithOpt, nil
		}
	}

	// WithOptionsClone is called only once
	// set withOptions and labelParser will not be used after that
	return &Middleware{
		m.name,
		m.before,
		m.modifyResponse,
		nil,
		m.impl,
		m.parent,
		m.children,
		false,
	}, nil
}

// TODO: check conflict or duplicates
func PatchReverseProxy(rpName string, rp *ReverseProxy, middlewaresMap map[string]OptionsRaw) (res E.NestedError) {
	middlewares := make([]*Middleware, 0, len(middlewaresMap))

	invalidM := E.NewBuilder("invalid middlewares")
	invalidOpts := E.NewBuilder("invalid options")
	defer func() {
		invalidM.Add(invalidOpts.Build())
		invalidM.To(&res)
	}()

	for name, opts := range middlewaresMap {
		m, ok := Get(name)
		if !ok {
			invalidM.Add(E.NotExist("middleware", name))
			continue
		}

		m, err := m.WithOptionsClone(opts)
		if err != nil {
			invalidOpts.Add(err.Subject(name))
			continue
		}
		middlewares = append(middlewares, m)
	}

	if invalidM.HasError() {
		return
	}

	patchReverseProxy(rpName, rp, middlewares)
	return
}

func patchReverseProxy(rpName string, rp *ReverseProxy, middlewares []*Middleware) {
	mid := BuildMiddlewareFromChain(rpName, middlewares)

	if mid.before != nil {
		ori := rp.ServeHTTP
		rp.ServeHTTP = func(w http.ResponseWriter, r *http.Request) {
			mid.before(ori, w, r)
		}
	}

	if mid.modifyResponse != nil {
		if rp.ModifyResponse != nil {
			ori := rp.ModifyResponse
			rp.ModifyResponse = func(res *http.Response) error {
				return errors.Join(mid.modifyResponse(res), ori(res))
			}
		} else {
			rp.ModifyResponse = mid.modifyResponse
		}
	}
}
