package common

import (
	"os"

	U "github.com/yusing/go-proxy/utils"
)

var (
	NoSchemaValidation = GetEnvBool("GOPROXY_NO_SCHEMA_VALIDATION")
	IsDebug            = GetEnvBool("GOPROXY_DEBUG")
	ProxyHTTPAddr      = GetEnv("GOPROXY_HTTP_ADDR", ":80")
	ProxyHTTPSAddr     = GetEnv("GOPROXY_HTTPS_ADDR", ":443")
	APIHTTPAddr        = GetEnv("GOPROXY_API_ADDR", "127.0.0.1:8888")
)

func GetEnvBool(key string) bool {
	return U.ParseBool(os.Getenv(key))
}

func GetEnv(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		value = defaultValue
	}
	return value
}
