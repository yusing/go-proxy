package utils

import (
	"encoding/json"
	"net/http"
)

func RespondJson(data any, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	j, err := json.Marshal(data)
	if err != nil {
		return err
	} else {
		w.Write(j)
	}
	return nil
}
