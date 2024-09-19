package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/error"
)

func HandleErr(w http.ResponseWriter, r *http.Request, origErr error, code ...int) {
	err := E.From(origErr).Subjectf("%s %s", r.Method, r.URL)
	logrus.WithField("module", "api").Error(err)
	if len(code) > 0 {
		http.Error(w, err.String(), code[0])
		return
	}
	http.Error(w, err.String(), http.StatusInternalServerError)
}

func ErrMissingKey(k string) error {
	return errors.New("missing key '" + k + "' in query or request body")
}

func ErrInvalidKey(k string) error {
	return errors.New("invalid key '" + k + "' in query or request body")
}

func ErrNotFound(k, v string) error {
	return fmt.Errorf("key %q with value %q not found", k, v)
}
