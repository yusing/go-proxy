package common

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var NoSchemaValidation = getEnvBool("GOPROXY_NO_SCHEMA_VALIDATION")
var IsDebug = getEnvBool("GOPROXY_DEBUG")

var LogLevel = func() logrus.Level {
	if IsDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return logrus.GetLevel()
}()

func getEnvBool(key string) bool {
	v := os.Getenv(key)
	return v == "1" || strings.ToLower(v) == "true" || strings.ToLower(v) == "yes" || strings.ToLower(v) == "on"
}
