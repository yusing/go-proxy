package middleware

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"
	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
	Error = E.Error

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
	CloneWithOptFunc   func(opts OptionsRaw) (*Middleware, E.Error)

	OptionsRaw = map[string]any

	Middleware struct {
		_ U.NoCopy

		zerolog.Logger

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

func (m *Middleware) WithOptionsClone(optsRaw OptionsRaw) (*Middleware, E.Error) {
	if m.withOptions != nil {
		m, err := m.withOptions(optsRaw)
		if err != nil {
			return nil, err
		}
		m.Logger = logger.With().Str("name", m.name).Logger()
		return m, nil
	}

	// WithOptionsClone is called only once
	// set withOptions and labelParser will not be used after that
	return &Middleware{
		Logger:         logger.With().Str("name", m.name).Logger(),
		name:           m.name,
		before:         m.before,
		modifyResponse: m.modifyResponse,
		impl:           m.impl,
		parent:         m.parent,
		children:       m.children,
	}, nil
}

func (m *Middleware) ModifyRequest(next http.HandlerFunc, w ResponseWriter, r *Request) {
	if m.before != nil {
		m.before(next, w, r)
	}
}

func (m *Middleware) ModifyResponse(resp *Response) error {
	if m.modifyResponse != nil {
		return m.modifyResponse(resp)
	}
	return nil
}

// TODO: check conflict or duplicates.
func createMiddlewares(middlewaresMap map[string]OptionsRaw) ([]*Middleware, E.Error) {
	middlewares := make([]*Middleware, 0, len(middlewaresMap))

	errs := E.NewBuilder("middlewares compile error")
	invalidOpts := E.NewBuilder("options compile error")

	for name, opts := range middlewaresMap {
		m, err := Get(name)
		if err != nil {
			errs.Add(err)
			continue
		}

		m, err = m.WithOptionsClone(opts)
		if err != nil {
			invalidOpts.Add(err.Subject(name))
			continue
		}
		middlewares = append(middlewares, m)
	}

	if invalidOpts.HasError() {
		errs.Add(invalidOpts.Error())
	}
	return middlewares, errs.Error()
}

func PatchReverseProxy(rpName string, rp *ReverseProxy, middlewaresMap map[string]OptionsRaw) (err E.Error) {
	var middlewares []*Middleware
	middlewares, err = createMiddlewares(middlewaresMap)
	if err != nil {
		return
	}
	patchReverseProxy(rpName, rp, middlewares)
	return
}

func patchReverseProxy(rpName string, rp *ReverseProxy, middlewares []*Middleware) {
	mid := BuildMiddlewareFromChain(rpName, middlewares)

	if mid.before != nil {
		ori := rp.HandlerFunc
		rp.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
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
