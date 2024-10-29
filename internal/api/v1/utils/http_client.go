package utils

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/yusing/go-proxy/internal/common"
)

var (
	httpClient = &http.Client{
		Timeout: common.ConnectionTimeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			ForceAttemptHTTP2: false,
			DialContext: (&net.Dialer{
				Timeout:   common.DialTimeout,
				KeepAlive: common.KeepAlive, // this is different from DisableKeepAlives
			}).DialContext,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	Get  = httpClient.Get
	Post = httpClient.Post
	Head = httpClient.Head
)
