package v1

import (
	"net"
	"net/http"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
)

func IsSiteHealthy(url string) bool {
	// try HEAD first
	// if HEAD is not allowed, try GET
	resp, err := U.Head(url)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil && resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
		_, err = U.Get(url)
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
