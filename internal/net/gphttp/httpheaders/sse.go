package httpheaders

import "net/http"

func IsSSE(h http.Header) bool {
	return h.Get("Content-Type") == "text/event-stream"
}
