package utils

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/yusing/go-proxy/internal/common"
)

var httpClient = &http.Client{
	Timeout: common.ConnectionTimeout,
	Transport: &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
		ForceAttemptHTTP2: true,
		DialContext: (&net.Dialer{
			Timeout:   common.DialTimeout,
			KeepAlive: common.KeepAlive, // this is different from DisableKeepAlives
		}).DialContext,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}
