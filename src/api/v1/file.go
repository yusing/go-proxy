package v1

import (
	"io"
	"net/http"
	"os"
	"path"

	U "github.com/yusing/go-proxy/api/v1/utils"
	"github.com/yusing/go-proxy/common"
	"github.com/yusing/go-proxy/config"
	E "github.com/yusing/go-proxy/error"
	"github.com/yusing/go-proxy/proxy/provider"
)

func GetFileContent(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if filename == "" {
		filename = common.ConfigFileName
	}
	content, err := os.ReadFile(path.Join(common.ConfigBasePath, filename))
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	w.Write(content)
}

func SetFileContent(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if filename == "" {
		U.HandleErr(w, r, U.ErrMissingKey("filename"), http.StatusBadRequest)
		return
	}
	content, err := E.Check(io.ReadAll(r.Body))
	if err.IsNotNil() {
		U.HandleErr(w, r, err)
		return
	}

	if filename == common.ConfigFileName {
		err = config.Validate(content)
	} else {
		err = provider.Validate(content)
	}

	if err.IsNotNil() {
		U.HandleErr(w, r, err, http.StatusBadRequest)
		return
	}

	err = E.From(os.WriteFile(path.Join(common.ConfigBasePath, filename), content, 0644))
	if err.IsNotNil() {
		U.HandleErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
