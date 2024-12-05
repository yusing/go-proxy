package http

import "net/http"

type ProxyResponse struct {
	*http.Response
	OriginalRequest *http.Request
}
