package utils

import (
	"github.com/santhosh-tekuri/jsonschema"
)

var schemaCompiler = func() *jsonschema.Compiler {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft7
	return c
}()

var schemaStorage = make(map[string]*jsonschema.Schema)

func GetSchema(path string) *jsonschema.Schema {
	if schema, ok := schemaStorage[path]; ok {
		return schema
	}
	schema := schemaCompiler.MustCompile(path)
	schemaStorage[path] = schema
	return schema
}
