package common

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	NoSchemaValidation = GetEnvBool("GOPROXY_NO_SCHEMA_VALIDATION", true)
	IsTest             = GetEnvBool("GOPROXY_TEST", false) || strings.HasSuffix(os.Args[0], ".test")
	IsDebug            = GetEnvBool("GOPROXY_DEBUG", IsTest)
	IsDebugSkipAuth    = GetEnvBool("GOPROXY_DEBUG_SKIP_AUTH", false)
	IsTrace            = GetEnvBool("GOPROXY_TRACE", false) && IsDebug

	ProxyHTTPAddr,
	ProxyHTTPHost,
	ProxyHTTPPort,
	ProxyHTTPURL = GetAddrEnv("GOPROXY_HTTP_ADDR", ":80", "http")

	ProxyHTTPSAddr,
	ProxyHTTPSHost,
	ProxyHTTPSPort,
	ProxyHTTPSURL = GetAddrEnv("GOPROXY_HTTPS_ADDR", ":443", "https")

	APIHTTPAddr,
	APIHTTPHost,
	APIHTTPPort,
	APIHTTPURL = GetAddrEnv("GOPROXY_API_ADDR", "127.0.0.1:8888", "http")

	APIJWTSecret    = decodeJWTKey(GetEnv("GOPROXY_API_JWT_SECRET", ""))
	APIJWTTokenTTL  = GetDurationEnv("GOPROXY_API_JWT_TOKEN_TTL", time.Hour)
	APIUser         = GetEnv("GOPROXY_API_USER", "admin")
	APIPasswordHash = HashPassword(GetEnv("GOPROXY_API_PASSWORD", "password"))
)

func init() {
	if APIJWTSecret == nil && GetArgs().Command == CommandStart {
		log.Warn().Msg("API JWT secret is empty, authentication is disabled")
	}
}

func GetEnvBool(key string, defaultValue bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		log.Fatal().Msgf("env %s: invalid boolean value: %s", key, value)
	}
	return b
}

func GetEnv(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		value = defaultValue
	}
	return value
}

func GetAddrEnv(key, defaultValue, scheme string) (addr, host, port, fullURL string) {
	addr = GetEnv(key, defaultValue)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatal().Msgf("env %s: invalid address: %s", key, addr)
	}
	if host == "" {
		host = "localhost"
	}
	fullURL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	return
}

func GetDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		log.Fatal().Msgf("env %s: invalid duration value: %s", key, value)
	}
	return d
}
