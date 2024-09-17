package common

import (
	"os"

	U "github.com/yusing/go-proxy/utils"
)

var NoSchemaValidation = getEnvBool("GOPROXY_NO_SCHEMA_VALIDATION")
var IsDebug = getEnvBool("GOPROXY_DEBUG")

func getEnvBool(key string) bool {
	return U.ParseBool(os.Getenv(key))
}
