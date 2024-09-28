package utils

import (
	"net"
	"net/http"

	"github.com/yusing/go-proxy/internal/common"
)

func IsSiteHealthy(url string) bool {
	// try HEAD first
	// if HEAD is not allowed, try GET
	resp, err := httpClient.Head(url)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil && resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
		_, err = httpClient.Get(url)
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
