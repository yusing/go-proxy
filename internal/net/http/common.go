package http

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
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           defaultDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConnsPerHost:   100,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	DefaultTransportNoTLS = func() *http.Transport {
		clone := DefaultTransport.Clone()
		clone.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return clone
	}()
)

const StaticFilePathPrefix = "/$gperrorpage/"
