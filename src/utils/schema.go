package utils

import (
	"github.com/santhosh-tekuri/jsonschema"
	"github.com/yusing/go-proxy/common"
)

var schemaCompiler = func() *jsonschema.Compiler {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft7
	return c
}()

var schemaStorage = make(map[string] *jsonschema.Schema)

func GetSchema(path string) *jsonschema.Schema {
	if common.NoSchemaValidation {
		panic("bug: GetSchema called when schema validation disabled")
	}
	if schema, ok := schemaStorage[path]; ok {
		return schema
	}
	schema := schemaCompiler.MustCompile(path)
	schemaStorage[path] = schema
	return schema
}