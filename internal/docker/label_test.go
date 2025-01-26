package docker_test

import (
	"testing"

	"github.com/yusing/go-proxy/internal/docker"
)

func BenchmarkParseLabels(b *testing.B) {
	for range b.N {
		_, _ = docker.ParseLabels(map[string]string{
			"proxy.a.host":   "localhost",
			"proxy.a.port":   "4444",
			"proxy.a.scheme": "http",
			"proxy.a.middlewares.request.hide_headers": "X-Header1,X-Header2",
		})
	}
}
