package v1

import (
	"net/http"

	"github.com/yusing/go-proxy/pkg"
)

func GetVersion(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(pkg.GetVersion()))
}
