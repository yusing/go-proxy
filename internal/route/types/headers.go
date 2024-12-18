package types

import (
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

func ValidateHTTPHeaders(headers map[string]string) (http.Header, E.Error) {
	h := make(http.Header)
	for k, v := range headers {
		vSplit := strutils.CommaSeperatedList(v)
		for _, header := range vSplit {
			h.Add(k, header)
		}
	}
	return h, nil
}
