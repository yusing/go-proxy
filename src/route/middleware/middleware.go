package middleware

import (
	"net/http"

	E "github.com/yusing/go-proxy/error"
	P "github.com/yusing/go-proxy/proxy"
)

type (
	ReverseProxy   = P.ReverseProxy
	ProxyRequest   = P.ProxyRequest
	Request        = http.Request
	Response       = http.Response
	ResponseWriter = http.ResponseWriter

	BeforeFunc         func(w ResponseWriter, r *Request) (continue_ bool)
	RewriteFunc        func(req *ProxyRequest)
	ModifyResponseFunc func(res *Response) error

	MiddlewareOptionsRaw map[string]string
	MiddlewareOptions    map[string]interface{}

	Middleware struct {
		name string

		before         BeforeFunc
		rewrite        RewriteFunc
		modifyResponse ModifyResponseFunc

		options         MiddlewareOptions
		validateOptions func(opts MiddlewareOptionsRaw) (MiddlewareOptions, E.NestedError)
	}
)

func (m *Middleware) Name() string {
	return m.name
}

func (m *Middleware) String() string {
	return m.name
}

func (m *Middleware) WithOptions(optsRaw MiddlewareOptionsRaw) (*Middleware, E.NestedError) {
	if len(optsRaw) == 0 {
		return m, nil
	}

	var opts MiddlewareOptions
	var err E.NestedError

	if m.validateOptions != nil {
		if opts, err = m.validateOptions(optsRaw); err != nil {
			return nil, err
		}
	}

	return &Middleware{
		name:           m.name,
		before:         m.before,
		rewrite:        m.rewrite,
		modifyResponse: m.modifyResponse,
		options:        opts,
	}, nil
}

// TODO: check conflict
func PatchReverseProxy(rp ReverseProxy, middlewares map[string]MiddlewareOptionsRaw) (out ReverseProxy, err E.NestedError) {
	out = rp

	befores := make([]BeforeFunc, 0, len(middlewares))
	rewrites := make([]RewriteFunc, 0, len(middlewares))
	modifyResponses := make([]ModifyResponseFunc, 0, len(middlewares))

	invalidM := E.NewBuilder("invalid middlewares")
	invalidOpts := E.NewBuilder("invalid options")
	defer invalidM.Add(invalidOpts.Build())
	defer invalidM.To(&err)

	for name, opts := range middlewares {
		m, ok := Get(name)
		if !ok {
			invalidM.Addf("%s", name)
			continue
		}
		m, err = m.WithOptions(opts)
		if err != nil {
			invalidOpts.Add(err.Subject(name))
			continue
		}
		if m.before != nil {
			befores = append(befores, m.before)
		}
		if m.rewrite != nil {
			rewrites = append(rewrites, m.rewrite)
		}
		if m.modifyResponse != nil {
			modifyResponses = append(modifyResponses, m.modifyResponse)
		}
	}

	if invalidM.HasError() {
		return
	}

	if len(befores) > 0 {
		rp.ServeHTTP = func(w ResponseWriter, r *Request) {
			for _, before := range befores {
				if !before(w, r) {
					return
				}
			}
			rp.ServeHTTP(w, r)
		}
	}
	if len(rewrites) > 0 {
		rp.Rewrite = func(req *ProxyRequest) {
			for _, rewrite := range rewrites {
				rewrite(req)
			}
		}
	}
	if len(modifyResponses) > 0 {
		rp.ModifyResponse = func(res *Response) error {
			for _, modifyResponse := range modifyResponses {
				if err := modifyResponse(res); err != nil {
					return err
				}
			}
			return nil
		}
	}

	return
}
