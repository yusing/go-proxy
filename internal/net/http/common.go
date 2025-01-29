package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	defaultDialer = net.Dialer{
		Timeout: 60 * time.Second,
	}
	DefaultTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           defaultDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true, // Prevent double compression
		ResponseHeaderTimeout: 60 * time.Second,
		WriteBufferSize:       16 * 1024, // 16KB
		ReadBufferSize:        16 * 1024, // 16KB
	}
	DefaultTransportNoTLS = func() *http.Transport {
		clone := DefaultTransport.Clone()
		clone.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return clone
	}()
)

const StaticFilePathPrefix = "/$gperrorpage/"
