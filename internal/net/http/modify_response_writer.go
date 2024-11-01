// Modified from Traefik Labs's MIT-licensed code (https://github.com/traefik/traefik/blob/master/pkg/middlewares/response_modifier.go)
// Copyright (c) 2020-2024 Traefik Labs

package http

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type (
	ModifyResponseFunc   func(*http.Response) error
	ModifyResponseWriter struct {
		w http.ResponseWriter
		r *http.Request

		headerSent bool
		code       int

		modifier    ModifyResponseFunc
		modified    bool
		modifierErr error
	}
)

func NewModifyResponseWriter(w http.ResponseWriter, r *http.Request, f ModifyResponseFunc) *ModifyResponseWriter {
	return &ModifyResponseWriter{
		w:        w,
		r:        r,
		modifier: f,
		code:     http.StatusOK,
	}
}

func (w *ModifyResponseWriter) WriteHeader(code int) {
	if w.headerSent {
		return
	}

	if code >= http.StatusContinue && code < http.StatusOK {
		w.w.WriteHeader(code)
	}

	defer func() {
		w.headerSent = true
		w.code = code
	}()

	if w.modifier == nil || w.modified {
		w.w.WriteHeader(code)
		return
	}

	resp := http.Response{
		Header:  w.w.Header(),
		Request: w.r,
	}

	if err := w.modifier(&resp); err != nil {
		w.modifierErr = fmt.Errorf("response modifier error: %w", err)
		w.w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.modified = true
	w.w.WriteHeader(code)
}

func (w *ModifyResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *ModifyResponseWriter) Write(b []byte) (int, error) {
	w.WriteHeader(w.code)
	if w.modifierErr != nil {
		return 0, w.modifierErr
	}
	return w.w.Write(b)
}

// Hijack hijacks the connection.
func (w *ModifyResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.w.(http.Hijacker); ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("not a hijacker: %T", w.w)
}

// Flush sends any buffered data to the client.
func (w *ModifyResponseWriter) Flush() {
	if flusher, ok := w.w.(http.Flusher); ok {
		flusher.Flush()
	}
}
