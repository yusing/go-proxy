package common

import (
	"fmt"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	U "github.com/yusing/go-proxy/internal/utils"
)

var (
	NoSchemaValidation = GetEnvBool("GOPROXY_NO_SCHEMA_VALIDATION")
	IsDebug            = GetEnvBool("GOPROXY_DEBUG")

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
)

func GetEnvBool(key string) bool {
	return U.ParseBool(os.Getenv(key))
}

func GetEnv(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		value = defaultValue
	}
	return value
}

func GetAddrEnv(key, defaultValue, scheme string) (addr, host, port, fullURL string) {
	addr = GetEnv(key, defaultValue)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		logrus.Fatalf("Invalid address: %s", addr)
	}
	if host == "" {
		host = "localhost"
	}
	fullURL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	return
}
