package utils

import (
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
)

func HandleErr(w http.ResponseWriter, r *http.Request, origErr error, code ...int) {
	if origErr == nil {
		return
	}
	LogError(r).Msg(origErr.Error())
	if len(code) > 0 {
		http.Error(w, origErr.Error(), code[0])
		return
	}
	http.Error(w, origErr.Error(), http.StatusInternalServerError)
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
