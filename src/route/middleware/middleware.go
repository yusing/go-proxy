package middleware

import (
	"net/http"

	D "github.com/yusing/go-proxy/docker"
	E "github.com/yusing/go-proxy/error"
	gpHTTP "github.com/yusing/go-proxy/http"
)

type (
	Error = E.NestedError

	ReverseProxy   = gpHTTP.ReverseProxy
	ProxyRequest   = gpHTTP.ProxyRequest
	Request        = http.Request
	Response       = http.Response
	ResponseWriter = http.ResponseWriter
	Header         = http.Header
	Cookie         = http.Cookie

	BeforeFunc         func(next http.Handler, w ResponseWriter, r *Request)
	RewriteFunc        func(req *ProxyRequest)
	ModifyResponseFunc func(res *Response) error
	CloneWithOptFunc   func(opts OptionsRaw, rp *ReverseProxy) (*Middleware, E.NestedError)

	OptionsRaw = map[string]any
	Options    any

	Middleware struct {
		name string

		before         BeforeFunc         // runs before ReverseProxy.ServeHTTP
		rewrite        RewriteFunc        // runs after ReverseProxy.Rewrite
		modifyResponse ModifyResponseFunc // runs after ReverseProxy.ModifyResponse

		transport http.RoundTripper

		withOptions    CloneWithOptFunc
		labelParserMap D.ValueParserMap
		impl           any
	}
)

func (m *Middleware) Name() string {
	return m.name
}

func (m *Middleware) String() string {
	return m.name
}

func (m *Middleware) WithOptionsClone(optsRaw OptionsRaw, rp *ReverseProxy) (*Middleware, E.NestedError) {
	if len(optsRaw) != 0 && m.withOptions != nil {
		if mWithOpt, err := m.withOptions(optsRaw, rp); err != nil {
			return nil, err
		} else {
			return mWithOpt, nil
		}
	}

	// WithOptionsClone is called only once
	// set withOptions and labelParser will not be used after that
	return &Middleware{m.name, m.before, m.rewrite, m.modifyResponse, m.transport, nil, nil, m.impl}, nil
}

// TODO: check conflict or duplicates
func PatchReverseProxy(rp *ReverseProxy, middlewares map[string]OptionsRaw) (res E.NestedError) {
	befores := make([]BeforeFunc, 0, len(middlewares))
	rewrites := make([]RewriteFunc, 0, len(middlewares))
	modifyResponses := make([]ModifyResponseFunc, 0, len(middlewares))

	invalidM := E.NewBuilder("invalid middlewares")
	invalidOpts := E.NewBuilder("invalid options")
	defer func() {
		invalidM.Add(invalidOpts.Build())
		invalidM.To(&res)
	}()

	for name, opts := range middlewares {
		m, ok := Get(name)
		if !ok {
			invalidM.Addf("%s", name)
			continue
		}

		m, err := m.WithOptionsClone(opts, rp)
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

	origServeHTTP := rp.ServeHTTP
	for i, before := range befores {
		if i < len(befores)-1 {
			rp.ServeHTTP = func(w ResponseWriter, r *Request) {
				before(rp.ServeHTTP, w, r)
			}
		} else {
			rp.ServeHTTP = func(w ResponseWriter, r *Request) {
				before(origServeHTTP, w, r)
			}
		}
	}

	if len(rewrites) > 0 {
		origRewrite := rp.Rewrite
		rp.Rewrite = func(req *ProxyRequest) {
			if origRewrite != nil {
				origRewrite(req)
			}
			for _, rewrite := range rewrites {
				rewrite(req)
			}
		}
	}

	if len(modifyResponses) > 0 {
		origModifyResponse := rp.ModifyResponse
		rp.ModifyResponse = func(res *Response) error {
			if origModifyResponse != nil {
				return origModifyResponse(res)
			}
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
