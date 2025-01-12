package utils

import (
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils/ansi"
)

// HandleErr logs the error and returns an error code to the client.
// If code is specified, it will be used as the HTTP status code; otherwise,
// http.StatusInternalServerError is used.
//
// The error is only logged but not returned to the client.
func HandleErr(w http.ResponseWriter, r *http.Request, err error, code ...int) {
	if err == nil {
		return
	}
	LogError(r).Msg(err.Error())
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
	// strip ANSI color codes added from Error.WithSubject
	http.Error(w, ansi.StripANSI(err.Error()), code[0])
}

func ErrMissingKey(k string) error {
	return E.New("missing key '" + k + "' in query or request body")
}

func ErrInvalidKey(k string) error {
	return E.New("invalid key '" + k + "' in query or request body")
}

func ErrNotFound(k, v string) error {
	return E.Errorf("key %q with value %q not found", k, v)
}
