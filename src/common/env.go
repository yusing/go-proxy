package common

import (
	"os"

	U "github.com/yusing/go-proxy/utils"
)

var (
	NoSchemaValidation = getEnvBool("GOPROXY_NO_SCHEMA_VALIDATION")
	IsDebug            = getEnvBool("GOPROXY_DEBUG")
	ProxyHTTPAddr      = getEnv("GOPROXY_HTTP_ADDR", ":80")
	ProxyHTTPSAddr     = getEnv("GOPROXY_HTTPS_ADDR", ":443")
	APIHTTPAddr        = getEnv("GOPROXY_API_ADDR", "127.0.0.1:8888")
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
