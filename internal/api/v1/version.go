package v1

import (
	"net/http"

	. "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/pkg"
)

func GetVersion(w http.ResponseWriter, r *http.Request) {
	WriteBody(w, []byte(pkg.GetVersion()))
}
