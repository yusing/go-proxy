package loadbalancer

import "net/http"

type DummyResponseWriter struct{}

func (w *DummyResponseWriter) Header() (_ http.Header) {
	return
}

func (w *DummyResponseWriter) Write([]byte) (_ int, _ error) {
	return
}

func (w *DummyResponseWriter) WriteHeader(int) {}
