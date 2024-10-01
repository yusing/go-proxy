package middleware

import (
	"net/http"
	"os"

	E "github.com/yusing/go-proxy/internal/error"
	"gopkg.in/yaml.v3"
)

func BuildMiddlewaresFromComposeFile(filePath string) (map[string]*Middleware, E.NestedError) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, E.FailWith("read middleware compose file", err)
	}
	return BuildMiddlewaresFromYAML(fileContent)
}

func BuildMiddlewaresFromYAML(data []byte) (middlewares map[string]*Middleware, outErr E.NestedError) {
	b := E.NewBuilder("middlewares compile errors")
	defer b.To(&outErr)

	var rawMap map[string][]map[string]any
	err := yaml.Unmarshal(data, &rawMap)
	if err != nil {
		b.Add(E.FailWith("toml unmarshal", err))
		return
	}
	middlewares = make(map[string]*Middleware)
	for name, defs := range rawMap {
		chainErr := E.NewBuilder(name)
		chain := make([]*Middleware, 0, len(defs))
		for i, def := range defs {
			if def["use"] == nil || def["use"].(string) == "" {
				chainErr.Add(E.Missing("use").Subjectf("%s.%d", name, i))
				continue
			}
			baseName := def["use"].(string)
			base, ok := Get(baseName)
			if !ok {
				chainErr.Add(E.NotExist("middleware", baseName).Subjectf("%s.%d", name, i))
				continue
			}
			delete(def, "use")
			m, err := base.WithOptionsClone(def)
			if err != nil {
				chainErr.Add(err.Subjectf("item%d", i))
				continue
			}
			chain = append(chain, m)
		}
		if chainErr.HasError() {
			b.Add(chainErr.Build())
		} else {
			name = name + "@file"
			middlewares[name] = BuildMiddlewareFromChain(name, chain)
		}
	}
	return
}

// TODO: check conflict or duplicates
func BuildMiddlewareFromChain(name string, chain []*Middleware) *Middleware {
	var (
		befores  []BeforeFunc
		rewrites []RewriteFunc
		modResps []ModifyResponseFunc
	)
	for _, m := range chain {
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

	m := &Middleware{name: name}
	if len(befores) > 0 {
		m.before = func(next http.Handler, w ResponseWriter, r *Request) {
			for _, before := range befores {
				before(next, w, r)
			}
		}
	}
	if len(rewrites) > 0 {
		m.rewrite = func(r *Request) {
			for _, rewrite := range rewrites {
				rewrite(r)
			}
		}
	}
	if len(modResps) > 0 {
		m.modifyResponse = func(res *Response) error {
			b := E.NewBuilder("errors in middleware %s", name)
			for _, mr := range modResps {
				b.AddE(mr(res))
			}
			return b.Build().Error()
		}
	}

	return m
}