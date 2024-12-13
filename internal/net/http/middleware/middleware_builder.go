package middleware

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"gopkg.in/yaml.v3"
)

var ErrMissingMiddlewareUse = E.New("missing middleware 'use' field")

func BuildMiddlewaresFromComposeFile(filePath string, eb *E.Builder) map[string]*Middleware {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		eb.Add(err)
		return nil
	}
	return BuildMiddlewaresFromYAML(path.Base(filePath), fileContent, eb)
}

func BuildMiddlewaresFromYAML(source string, data []byte, eb *E.Builder) map[string]*Middleware {
	var rawMap map[string][]map[string]any
	err := yaml.Unmarshal(data, &rawMap)
	if err != nil {
		eb.Add(err)
		return nil
	}
	middlewares := make(map[string]*Middleware)
	for name, defs := range rawMap {
		chain, err := BuildMiddlewareFromChainRaw(name, defs)
		if err != nil {
			eb.Add(err.Subject(source))
		} else {
			middlewares[name+"@file"] = chain
		}
	}
	return middlewares
}

func BuildMiddlewareFromChainRaw(name string, defs []map[string]any) (*Middleware, E.Error) {
	chainErr := E.NewBuilder("")
	chain := make([]*Middleware, 0, len(defs))
	for i, def := range defs {
		if def["use"] == nil || def["use"] == "" {
			chainErr.Add(ErrMissingMiddlewareUse.Subjectf("%s[%d]", name, i))
			continue
		}
		baseName := def["use"].(string)
		base, err := Get(baseName)
		if err != nil {
			chainErr.Add(err.Subjectf("%s[%d]", name, i))
			continue
		}
		delete(def, "use")
		m, err := base.WithOptionsClone(def)
		if err != nil {
			chainErr.Add(err.Subjectf("%s[%d]", name, i))
			continue
		}
		m.name = fmt.Sprintf("%s[%d]", name, i)
		chain = append(chain, m)
	}
	if chainErr.HasError() {
		return nil, chainErr.Error()
	}
	return BuildMiddlewareFromChain(name, chain), nil
}

// TODO: check conflict or duplicates.
func BuildMiddlewareFromChain(name string, chain []*Middleware) *Middleware {
	m := &Middleware{name: name, children: chain}

	var befores []*Middleware
	var modResps []*Middleware

	for _, comp := range chain {
		if comp.before != nil {
			befores = append(befores, comp)
		}
		if comp.modifyResponse != nil {
			modResps = append(modResps, comp)
		}
		comp.parent = m
	}

	if len(befores) > 0 {
		m.before = buildBefores(befores)
	}
	if len(modResps) > 0 {
		m.modifyResponse = func(res *Response) error {
			errs := E.NewBuilder("modify response errors")
			for _, mr := range modResps {
				if err := mr.modifyResponse(res); err != nil {
					errs.Add(E.From(err).Subject(mr.name))
				}
			}
			return errs.Error()
		}
	}

	if common.IsDebug {
		m.EnableTrace()
		m.AddTracef("middleware created")
	}
	return m
}

func buildBefores(befores []*Middleware) BeforeFunc {
	if len(befores) == 1 {
		return befores[0].before
	}
	nextBefores := buildBefores(befores[1:])
	return func(next http.HandlerFunc, w ResponseWriter, r *Request) {
		befores[0].before(func(w ResponseWriter, r *Request) {
			nextBefores(next, w, r)
		}, w, r)
	}
}
