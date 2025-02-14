package utils

import (
	"context"
	"errors"
	"net/http"
	"syscall"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/httpheaders"
	"github.com/yusing/go-proxy/internal/utils/strutils/ansi"
)

// HandleErr logs the error and returns an error code to the client.
// If code is specified, it will be used as the HTTP status code; otherwise,
// http.StatusInternalServerError is used.
//
// The error is only logged but not returned to the client.
func HandleErr(w http.ResponseWriter, r *http.Request, err error, code ...int) {
	switch {
	case err == nil,
		errors.Is(err, context.Canceled),
		errors.Is(err, syscall.EPIPE),
		errors.Is(err, syscall.ECONNRESET):
		return
	}
	LogError(r).Msg(err.Error())
	if httpheaders.IsWebsocket(r.Header) {
		return
	}
	if len(code) == 0 {
		code = []int{http.StatusInternalServerError}
	}
	http.Error(w, http.StatusText(code[0]), code[0])
}

// RespondError returns error details to the client.
// If code is specified, it will be used as the HTTP status code; otherwise,
// http.StatusBadRequest is used.
func RespondError(w http.ResponseWriter, err error, code ...int) {
	if len(code) == 0 {
		code = []int{http.StatusBadRequest}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.Error(w, ansi.StripANSI(err.Error()), code[0])
}

func Errorf(format string, args ...any) error {
	return E.Errorf(format, args...)
}

func ErrMissingKey(k string) error {
	return E.New(k + " is required")
}

func ErrInvalidKey(k string) error {
	return E.New(k + " is invalid")
}

func ErrAlreadyExists(k, v string) error {
	return E.Errorf("%s %q already exists", k, v)
}

func ErrNotFound(k, v string) error {
	return E.Errorf("%s %q not found", k, v)
}
