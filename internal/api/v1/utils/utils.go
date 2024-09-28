package utils

import (
	"encoding/json"
	"net/http"
)

func RespondJson(w http.ResponseWriter, data any, code ...int) error {
	if len(code) > 0 {
		w.WriteHeader(code[0])
	}
	w.Header().Set("Content-Type", "application/json")
	j, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	} else {
		w.Write(j)
	}
	return nil
}
