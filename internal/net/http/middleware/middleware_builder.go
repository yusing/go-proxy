package middleware

import (
	"fmt"
	"net/http"
	"os"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"gopkg.in/yaml.v3"
)

func BuildMiddlewaresFromComposeFile(filePath string) (map[string]*Middleware, E.Error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, E.FailWith("read middleware compose file", err)
	}
	return BuildMiddlewaresFromYAML(fileContent)
}

func BuildMiddlewaresFromYAML(data []byte) (middlewares map[string]*Middleware, outErr E.Error) {
	b := E.NewBuilder("middlewares compile errors")
	defer b.To(&outErr)

	var rawMap map[string][]map[string]any
	err := yaml.Unmarshal(data, &rawMap)
	if err != nil {
		b.Add(E.FailWith("yaml unmarshal", err))
		return
	}
	middlewares = make(map[string]*Middleware)
	for name, defs := range rawMap {
		chainErr := E.NewBuilder("%s", name)
		chain := make([]*Middleware, 0, len(defs))
		for i, def := range defs {
			if def["use"] == nil || def["use"] == "" {
				chainErr.Add(E.Missing("use").Subjectf(".%d", i))
				continue
			}
			baseName := def["use"].(string)
			base, ok := Get(baseName)
			if !ok {
				base, ok = middlewares[baseName]
				if !ok {
					chainErr.Add(E.NotExist("middleware", baseName).Subjectf(".%d", i))
					continue
				}
			}
			delete(def, "use")
			m, err := base.WithOptionsClone(def)
			if err != nil {
				chainErr.Add(err.Subjectf("item%d", i))
				continue
			}
			m.name = fmt.Sprintf("%s[%d]", name, i)
			chain = append(chain, m)
		}
		if chainErr.HasError() {
			b.Add(chainErr.Build())
		} else {
			middlewares[name+"@file"] = BuildMiddlewareFromChain(name, chain)
		}
	}
	return
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
			b := E.NewBuilder("errors in middleware")
			for _, mr := range modResps {
				b.Add(E.From(mr.modifyResponse(res)).Subject(mr.name))
			}
			return b.Build().Error()
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
