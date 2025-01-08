package middleware

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/http/reverseproxy"
	"github.com/yusing/go-proxy/internal/utils"
)

type (
	Error = E.Error

	ReverseProxy = reverseproxy.ReverseProxy
	ProxyRequest = reverseproxy.ProxyRequest

	ImplNewFunc = func() any
	OptionsRaw  = map[string]any

	Middleware struct {
		name      string
		construct ImplNewFunc
		impl      any
	}

	RequestModifier interface {
		before(w http.ResponseWriter, r *http.Request) (proceed bool)
	}
	ResponseModifier     interface{ modifyResponse(r *http.Response) error }
	MiddlewareWithSetup  interface{ setup() }
	MiddlewareFinalizer  interface{ finalize() }
	MiddlewareWithTracer interface {
		enableTrace()
		getTracer() *Tracer
	}
)

func NewMiddleware[ImplType any]() *Middleware {
	// type check
	switch any(new(ImplType)).(type) {
	case RequestModifier:
	case ResponseModifier:
	default:
		panic("must implement RequestModifier or ResponseModifier")
	}
	return &Middleware{
		name:      strings.ToLower(reflect.TypeFor[ImplType]().Name()),
		construct: func() any { return new(ImplType) },
	}
}

func (m *Middleware) enableTrace() {
	if tracer, ok := m.impl.(MiddlewareWithTracer); ok {
		tracer.enableTrace()
		logging.Debug().Msgf("middleware %s enabled trace", m.name)
	}
}

func (m *Middleware) getTracer() *Tracer {
	if tracer, ok := m.impl.(MiddlewareWithTracer); ok {
		return tracer.getTracer()
	}
	return nil
}

func (m *Middleware) setParent(parent *Middleware) {
	if tracer := m.getTracer(); tracer != nil {
		tracer.SetParent(parent.getTracer())
	}
}

func (m *Middleware) setup() {
	if setup, ok := m.impl.(MiddlewareWithSetup); ok {
		setup.setup()
	}
}

func (m *Middleware) apply(optsRaw OptionsRaw) E.Error {
	if len(optsRaw) == 0 {
		return nil
	}
	return utils.Deserialize(optsRaw, m.impl)
}

func (m *Middleware) finalize() {
	if finalizer, ok := m.impl.(MiddlewareFinalizer); ok {
		finalizer.finalize()
	}
}

func (m *Middleware) New(optsRaw OptionsRaw) (*Middleware, E.Error) {
	if m.construct == nil { // likely a middleware from compose
		if len(optsRaw) != 0 {
			return nil, E.New("additional options not allowed for middleware ").Subject(m.name)
		}
		return m, nil
	}
	mid := &Middleware{name: m.name, impl: m.construct()}
	mid.setup()
	if err := mid.apply(optsRaw); err != nil {
		return nil, err
	}
	mid.finalize()
	return mid, nil
}

func (m *Middleware) Name() string {
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

func (m *Middleware) ModifyRequest(next http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	if exec, ok := m.impl.(RequestModifier); ok {
		if proceed := exec.before(w, r); !proceed {
			return
		}
	}
	next(w, r)
}

func (m *Middleware) ModifyResponse(resp *http.Response) error {
	if exec, ok := m.impl.(ResponseModifier); ok {
		return exec.modifyResponse(resp)
	}
	return nil
}

func (m *Middleware) ServeHTTP(next http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	if exec, ok := m.impl.(ResponseModifier); ok {
		w = gphttp.NewModifyResponseWriter(w, r, func(resp *http.Response) error {
			return exec.modifyResponse(resp)
		})
	}
	if exec, ok := m.impl.(RequestModifier); ok {
		if proceed := exec.before(w, r); !proceed {
			return
		}
	}
	next(w, r)
}

// TODO: check conflict or duplicates.
func compileMiddlewares(middlewaresMap map[string]OptionsRaw) ([]*Middleware, E.Error) {
	middlewares := make([]*Middleware, 0, len(middlewaresMap))

	errs := E.NewBuilder("middlewares compile error")
	invalidOpts := E.NewBuilder("options compile error")

	for name, opts := range middlewaresMap {
		m, err := Get(name)
		if err != nil {
			errs.Add(err)
			continue
		}

		m, err = m.New(opts)
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

func PatchReverseProxy(rp *ReverseProxy, middlewaresMap map[string]OptionsRaw) (err E.Error) {
	var middlewares []*Middleware
	middlewares, err = compileMiddlewares(middlewaresMap)
	if err != nil {
		return
	}
	patchReverseProxy(rp, middlewares)
	return
}

func patchReverseProxy(rp *ReverseProxy, middlewares []*Middleware) {
	middlewares = append([]*Middleware{newSetUpstreamHeaders(rp)}, middlewares...)

	mid := NewMiddlewareChain(rp.TargetName, middlewares)

	if before, ok := mid.impl.(RequestModifier); ok {
		next := rp.HandlerFunc
		rp.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			if proceed := before.before(w, r); proceed {
				next(w, r)
			}
		}
	}

	if mr, ok := mid.impl.(ResponseModifier); ok {
		if rp.ModifyResponse != nil {
			ori := rp.ModifyResponse
			rp.ModifyResponse = func(res *http.Response) error {
				if err := mr.modifyResponse(res); err != nil {
					return err
				}
				return ori(res)
			}
		} else {
			rp.ModifyResponse = mr.modifyResponse
		}
	}
}
