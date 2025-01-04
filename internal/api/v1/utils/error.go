package utils

import (
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils/ansi"
)

// HandleErr logs the error and returns an HTTP error response to the client.
// If code is specified, it will be used as the HTTP status code; otherwise,
// http.StatusInternalServerError is used.
//
// The error is only logged but not returned to the client.
func HandleErr(w http.ResponseWriter, r *http.Request, origErr error, code ...int) {
	if origErr == nil {
		return
	}
	LogError(r).Msg(origErr.Error())
	statusCode := http.StatusInternalServerError
	if len(code) > 0 {
		statusCode = code[0]
	}
	http.Error(w, http.StatusText(statusCode), statusCode)
}

func RespondError(w http.ResponseWriter, err error, code ...int) {
	if len(code) > 0 {
		w.WriteHeader(code[0])
	}
	WriteBody(w, []byte(ansi.StripANSI(err.Error())))
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
