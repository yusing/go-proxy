package v1

import (
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/config"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/route/provider"
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
	U.WriteBody(w, content)
}

func SetFileContent(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if filename == "" {
		U.HandleErr(w, r, U.ErrMissingKey("filename"), http.StatusBadRequest)
		return
	}
	content, err := io.ReadAll(r.Body)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}

	var validateErr E.NestedError
	if filename == common.ConfigFileName {
		validateErr = config.Validate(content)
	} else if !strings.HasPrefix(filename, path.Base(common.MiddlewareComposeBasePath)) {
		validateErr = provider.Validate(content)
	}

	if validateErr != nil {
		U.RespondJSON(w, r, validateErr.JSONObject(), http.StatusBadRequest)
		return
	}

	err = os.WriteFile(path.Join(common.ConfigBasePath, filename), content, 0o644)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
