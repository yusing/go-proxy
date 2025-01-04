package utils

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/utils/strutils/ansi"
)

func WriteBody(w http.ResponseWriter, body []byte) {
	if _, err := w.Write(body); err != nil {
		HandleErr(w, nil, err)
	}
}

func RespondJSON(w http.ResponseWriter, r *http.Request, data any, code ...int) (canProceed bool) {
	if len(code) > 0 {
		w.WriteHeader(code[0])
	}
	w.Header().Set("Content-Type", "application/json")
	var j []byte
	var err error

	switch data := data.(type) {
	case string:
		j = []byte(fmt.Sprintf("%q", data))
	case []byte:
		j = data
	case error:
		j, err = json.Marshal(ansi.StripANSI(data.Error()))
	default:
		j, err = json.MarshalIndent(data, "", "  ")
	}

	if err != nil {
		logging.Panic().Err(err).Msg("failed to marshal json")
		return false
	}

	_, err = w.Write(j)
	if err != nil {
		HandleErr(w, r, err)
		return false
	}
	return true
}
