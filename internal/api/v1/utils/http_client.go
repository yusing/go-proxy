package utils

import (
	"bytes"
	"crypto/tls"
	"io"
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

func FetchAPI(method, endpoint string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, "http://localhost"+common.APIHTTPAddr+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}
	return httpClient.Do(req)
}
