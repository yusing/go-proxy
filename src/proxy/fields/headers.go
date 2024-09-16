package fields

import (
	"net/http"
	"strings"

	E "github.com/yusing/go-proxy/error"
)

func NewHTTPHeaders(headers map[string]string) (http.Header, E.NestedError) {
	h := make(http.Header)
	for k, v := range headers {
		vSplit := strings.Split(v, ",")
		for _, header := range vSplit {
			h.Add(k, strings.TrimSpace(header))
		}
	}
	return h, E.Nil()
}