package utils

import (
	"encoding/json"
	"net/http"
)

func WriteBody(w http.ResponseWriter, body []byte) {
	if _, err := w.Write(body); err != nil {
		HandleErr(w, nil, err)
	}
}

func RespondJSON(w http.ResponseWriter, r *http.Request, data any, code ...int) bool {
	if len(code) > 0 {
		w.WriteHeader(code[0])
	}
	w.Header().Set("Content-Type", "application/json")
	j, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		HandleErr(w, r, err)
		return false
	}
	_, err = w.Write(j)
	if err != nil {
		HandleErr(w, r, err)
		return false
	}
	return true
}
