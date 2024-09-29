package middleware

import (
	"net/http"

	D "github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
	gpHTTP "github.com/yusing/go-proxy/internal/net/http"
	U "github.com/yusing/go-proxy/internal/utils"
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
	RewriteFunc        func(req *Request)
	ModifyResponseFunc func(resp *Response) error
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

var Deserialize = U.Deserialize

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
	modResps := make([]ModifyResponseFunc, 0, len(middlewares))

	invalidM := E.NewBuilder("invalid middlewares")
	invalidOpts := E.NewBuilder("invalid options")
	defer func() {
		invalidM.Add(invalidOpts.Build())
		invalidM.To(&res)
	}()

	for name, opts := range middlewares {
		m, ok := Get(name)
		if !ok {
			invalidM.Add(E.NotExist("middleware", name))
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
			modResps = append(modResps, m.modifyResponse)
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
		origServeHTTP = rp.ServeHTTP
		rp.ServeHTTP = func(w http.ResponseWriter, r *http.Request) {
			for _, rewrite := range rewrites {
				rewrite(r)
			}
			origServeHTTP(w, r)
		}
	}

	if len(modResps) > 0 {
		if rp.ModifyResponse != nil {
			modResps = append([]ModifyResponseFunc{rp.ModifyResponse}, modResps...)
		}
		rp.ModifyResponse = func(res *Response) error {
			b := E.NewBuilder("errors in middleware ModifyResponse")
			for _, mr := range modResps {
				b.AddE(mr(res))
			}
			return b.Build().Error()
		}
	}

	return
}
