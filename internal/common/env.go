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
	prefixes = []string{"GODOXY_", "GOPROXY_", ""}

	IsTest          = GetEnvBool("TEST", false) || strings.HasSuffix(os.Args[0], ".test")
	IsDebug         = GetEnvBool("DEBUG", IsTest)
	IsDebugSkipAuth = GetEnvBool("DEBUG_SKIP_AUTH", false)
	IsTrace         = GetEnvBool("TRACE", false) && IsDebug
	IsProduction    = !IsTest && !IsDebug

	ProxyHTTPAddr,
	ProxyHTTPHost,
	ProxyHTTPPort,
	ProxyHTTPURL = GetAddrEnv("HTTP_ADDR", ":80", "http")

	ProxyHTTPSAddr,
	ProxyHTTPSHost,
	ProxyHTTPSPort,
	ProxyHTTPSURL = GetAddrEnv("HTTPS_ADDR", ":443", "https")

	APIHTTPAddr,
	APIHTTPHost,
	APIHTTPPort,
	APIHTTPURL = GetAddrEnv("API_ADDR", "127.0.0.1:8888", "http")

	MetricsHTTPAddr,
	MetricsHTTPHost,
	MetricsHTTPPort,
	MetricsHTTPURL = GetAddrEnv("PROMETHEUS_ADDR", "", "http")
	PrometheusEnabled = MetricsHTTPURL != ""

	APIJWTSecret    = decodeJWTKey(GetEnvString("API_JWT_SECRET", ""))
	APIJWTTokenTTL  = GetDurationEnv("API_JWT_TOKEN_TTL", time.Hour)
	APIUser         = GetEnvString("API_USER", "admin")
	APIPasswordHash = HashPassword(GetEnvString("API_PASSWORD", "password"))
)

func GetEnv[T any](key string, defaultValue T, parser func(string) (T, error)) T {
	var value string
	var ok bool
	for _, prefix := range prefixes {
		value, ok = os.LookupEnv(prefix + key)
		if ok && value != "" {
			break
		}
	}
	if !ok || value == "" {
		return defaultValue
	}
	parsed, err := parser(value)
	if err == nil {
		return parsed
	}
	log.Fatal().Err(err).Msgf("env %s: invalid %T value: %s", key, parsed, value)
	return defaultValue
}

func GetEnvString(key string, defaultValue string) string {
	return GetEnv(key, defaultValue, func(s string) (string, error) {
		return s, nil
	})
}

func GetEnvBool(key string, defaultValue bool) bool {
	return GetEnv(key, defaultValue, strconv.ParseBool)
}

func GetAddrEnv(key, defaultValue, scheme string) (addr, host, port, fullURL string) {
	addr = GetEnvString(key, defaultValue)
	if addr == "" {
		return
	}
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
	return GetEnv(key, defaultValue, time.ParseDuration)
}

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
	prefixes = []string{"GODOXY_", "GOPROXY_", ""}

	IsTest          = GetEnvBool("TEST", false) || strings.HasSuffix(os.Args[0], ".test")
	IsDebug         = GetEnvBool("DEBUG", IsTest)
	IsDebugSkipAuth = GetEnvBool("DEBUG_SKIP_AUTH", false)
	IsTrace         = GetEnvBool("TRACE", false) && IsDebug
	IsProduction    = !IsTest && !IsDebug

	ProxyHTTPAddr,
	ProxyHTTPHost,
	ProxyHTTPPort,
	ProxyHTTPURL = GetAddrEnv("HTTP_ADDR", ":80", "http")

	ProxyHTTPSAddr,
	ProxyHTTPSHost,
	ProxyHTTPSPort,
	ProxyHTTPSURL = GetAddrEnv("HTTPS_ADDR", ":443", "https")

	APIHTTPAddr,
	APIHTTPHost,
	APIHTTPPort,
	APIHTTPURL = GetAddrEnv("API_ADDR", "127.0.0.1:8888", "http")

	MetricsHTTPAddr,
	MetricsHTTPHost,
	MetricsHTTPPort,
	MetricsHTTPURL = GetAddrEnv("PROMETHEUS_ADDR", "", "http")
	PrometheusEnabled = MetricsHTTPURL != ""

	APIJWTSecret    = decodeJWTKey(GetEnvString("API_JWT_SECRET", ""))
	APIJWTTokenTTL  = GetDurationEnv("API_JWT_TOKEN_TTL", time.Hour)
	APIUser         = GetEnvString("API_USER", "admin")
	APIPasswordHash = HashPassword(GetEnvString("API_PASSWORD", "password"))

	// OIDC Configuration
	OIDCIssuerURL    = GetEnvString("OIDC_ISSUER_URL", "")
	OIDCClientID     = GetEnvString("OIDC_CLIENT_ID", "")
	OIDCClientSecret = GetEnvString("OIDC_CLIENT_SECRET", "")
	OIDCRedirectURL  = GetEnvString("OIDC_REDIRECT_URL", "")
)

func GetEnv[T any](key string, defaultValue T, parser func(string) (T, error)) T {
	var value string
	var ok bool
	for _, prefix := range prefixes {
		value, ok = os.LookupEnv(prefix + key)
		if ok && value != "" {
			break
		}
	}
	if !ok || value == "" {
		return defaultValue
	}
	parsed, err := parser(value)
	if err == nil {
		return parsed
	}
	log.Fatal().Err(err).Msgf("env %s: invalid %T value: %s", key, parsed, value)
	return defaultValue
}

func GetEnvString(key string, defaultValue string) string {
	return GetEnv(key, defaultValue, func(s string) (string, error) {
		return s, nil
	})
}

func GetEnvBool(key string, defaultValue bool) bool {
	return GetEnv(key, defaultValue, strconv.ParseBool)
}

func GetAddrEnv(key, defaultValue, scheme string) (addr, host, port, fullURL string) {
	addr = GetEnvString(key, defaultValue)
	if addr == "" {
		return
	}
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
	return GetEnv(key, defaultValue, time.ParseDuration)
}
