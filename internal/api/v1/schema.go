package v1

import (
	"net/http"
	"os"
	"path"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
)

func GetSchemaFile(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if filename == "" {
		U.RespondError(w, U.ErrMissingKey("filename"), http.StatusBadRequest)
	}
	content, err := os.ReadFile(path.Join(common.SchemasBasePath, filename))
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	U.WriteBody(w, content)
}
