package common

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	defaultDialer = net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}
	DefaultTransport = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         defaultDialer.DialContext,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     90 * time.Second,
	}
	DefaultTransportNoTLS = func() *http.Transport {
		var clone = DefaultTransport.Clone()
		clone.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return clone
	}()
)

const StaticFilePathPrefix = "/$gperrorpage/"
