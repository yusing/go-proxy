package common

import (
	"os"
	"strings"
)

var NoSchemaValidation = getEnvBool("GOPROXY_NO_SCHEMA_VALIDATION")
var IsDebug = getEnvBool("GOPROXY_DEBUG")

func getEnvBool(key string) bool {
	switch strings.ToLower(os.Getenv(key)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
