package utils

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yusing/go-proxy/internal/logging"
)

func WriteBody(w http.ResponseWriter, body []byte) {
	if _, err := w.Write(body); err != nil {
		logging.Err(err).Msg("failed to write body")
	}
}

func RespondJSON(w http.ResponseWriter, r *http.Request, data any, code ...int) (canProceed bool) {
	if len(code) > 0 {
		w.WriteHeader(code[0])
	}
	w.Header().Set("Content-Type", "application/json")
	var err error

	switch data := data.(type) {
	case string:
		_, err = w.Write([]byte(fmt.Sprintf("%q", data)))
	case []byte:
		_, err = w.Write(data)
	default:
		err = json.NewEncoder(w).Encode(data)
	}

	if err != nil {
		HandleErr(w, r, err)
		return false
	}
	return true
}
