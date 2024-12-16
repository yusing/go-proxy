package middleware

import (
	"fmt"
	"os"
	"path"

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
		m, err := base.New(def)
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
	return NewMiddlewareChain(name, chain), nil
}
