package common

import (
	"os"

	U "github.com/yusing/go-proxy/utils"
)

var (
	NoSchemaValidation = getEnvBool("GOPROXY_NO_SCHEMA_VALIDATION")
	IsDebug            = getEnvBool("GOPROXY_DEBUG")
	ProxyHTTPPort      = ":" + getEnv("GOPROXY_HTTP_PORT", "80")
	ProxyHTTPSPort     = ":" + getEnv("GOPROXY_HTTPS_PORT", "443")
	APIHTTPPort        = ":" + getEnv("GOPROXY_API_PORT", "8888")
)

func getEnvBool(key string) bool {
	return U.ParseBool(os.Getenv(key))
}

func getEnv(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		value = defaultValue
	}
	return value
}
