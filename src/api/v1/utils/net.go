package utils

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
)

func IsSiteHealthy(url string) bool {
	// try HEAD first
	// if HEAD is not allowed, try GET
	resp, err := HttpClient.Head(url)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil && resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
		_, err = HttpClient.Get(url)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return err == nil
}

func IsStreamHealthy(scheme, address string) bool {
	conn, err := net.DialTimeout(scheme, address, common.DialTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func ReloadServer() E.NestedError {
	resp, err := HttpClient.Post(fmt.Sprintf("http://localhost%v/reload", common.APIHTTPPort), "", nil)
	if err != nil {
		return E.From(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return E.Failure("server reload").Subjectf("status code: %v", resp.StatusCode)
	}
	return nil
}

var HttpClient = &http.Client{
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
