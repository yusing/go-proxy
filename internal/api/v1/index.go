package v1

import (
	"net/http"

	. "github.com/yusing/go-proxy/internal/api/v1/utils"
)

func Index(w http.ResponseWriter, r *http.Request) {
	WriteBody(w, []byte("API ready"))
}
