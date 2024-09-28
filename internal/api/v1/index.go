package v1

import "net/http"

func Index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("API ready"))
}
