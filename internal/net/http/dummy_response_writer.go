package http

import "net/http"

type DummyResponseWriter struct{}

func (w DummyResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (w DummyResponseWriter) Write([]byte) (_ int, _ error) {
	return
}

func (w DummyResponseWriter) WriteHeader(int) {}
