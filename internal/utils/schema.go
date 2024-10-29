package utils

import (
	"sync"

	"github.com/santhosh-tekuri/jsonschema"
)

var (
	schemaCompiler = jsonschema.NewCompiler()
	schemaStorage  = make(map[string]*jsonschema.Schema)
	schemaMu       sync.Mutex
)

func GetSchema(path string) *jsonschema.Schema {
	if schema, ok := schemaStorage[path]; ok {
		return schema
	}
	schemaMu.Lock()
	defer schemaMu.Unlock()
	if schema, ok := schemaStorage[path]; ok {
		return schema
	}
	schema := schemaCompiler.MustCompile(path)
	schemaStorage[path] = schema
	return schema
}
