package middleware

import (
	_ "embed"
	"encoding/json"
	"testing"

	E "github.com/yusing/go-proxy/internal/error"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

//go:embed test_data/middleware_compose.yml
var testMiddlewareCompose []byte

func TestBuild(t *testing.T) {
	middlewares, err := BuildMiddlewaresFromYAML(testMiddlewareCompose)
	ExpectNoError(t, err.Error())
	_, err = E.Check(json.MarshalIndent(middlewares, "", "  "))
	ExpectNoError(t, err.Error())
	// t.Log(string(data))
	// TODO: test
}
