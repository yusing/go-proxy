package v1

import (
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	config "github.com/yusing/go-proxy/internal/config/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
	"github.com/yusing/go-proxy/internal/route/provider"
)

type FileType string

const (
	FileTypeConfig     FileType = "config"
	FileTypeProvider   FileType = "provider"
	FileTypeMiddleware FileType = "middleware"
)

func fileType(file string) FileType {
	switch {
	case strings.HasPrefix(path.Base(file), "config."):
		return FileTypeConfig
	case strings.HasPrefix(file, common.MiddlewareComposeBasePath):
		return FileTypeMiddleware
	}
	return FileTypeProvider
}

func (t FileType) IsValid() bool {
	switch t {
	case FileTypeConfig, FileTypeProvider, FileTypeMiddleware:
		return true
	}
	return false
}

func (t FileType) GetPath(filename string) string {
	if t == FileTypeMiddleware {
		return path.Join(common.MiddlewareComposeBasePath, filename)
	}
	return path.Join(common.ConfigBasePath, filename)
}

func getArgs(r *http.Request) (fileType FileType, filename string, err error) {
	fileType = FileType(r.PathValue("type"))
	if !fileType.IsValid() {
		err = U.ErrInvalidKey("type")
		return
	}
	filename = r.PathValue("filename")
	if filename == "" {
		err = U.ErrMissingKey("filename")
	}
	return
}

func GetFileContent(w http.ResponseWriter, r *http.Request) {
	fileType, filename, err := getArgs(r)
	if err != nil {
		U.RespondError(w, err, http.StatusBadRequest)
		return
	}
	content, err := os.ReadFile(fileType.GetPath(filename))
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	U.WriteBody(w, content)
}

func validateFile(fileType FileType, content []byte) error {
	switch fileType {
	case FileTypeConfig:
		return config.Validate(content)
	case FileTypeMiddleware:
		errs := E.NewBuilder("middleware errors")
		middleware.BuildMiddlewaresFromYAML("", content, errs)
		return errs.Error()
	}
	return provider.Validate(content)
}

func ValidateFile(w http.ResponseWriter, r *http.Request) {
	fileType := FileType(r.PathValue("type"))
	if !fileType.IsValid() {
		U.RespondError(w, U.ErrInvalidKey("type"), http.StatusBadRequest)
		return
	}
	content, err := io.ReadAll(r.Body)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	r.Body.Close()
	err = validateFile(fileType, content)
	if err != nil {
		U.RespondError(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func SetFileContent(w http.ResponseWriter, r *http.Request) {
	fileType, filename, err := getArgs(r)
	if err != nil {
		U.RespondError(w, err, http.StatusBadRequest)
		return
	}
	content, err := io.ReadAll(r.Body)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}

	if valErr := validateFile(fileType, content); valErr != nil {
		U.RespondError(w, valErr, http.StatusBadRequest)
		return
	}

	err = os.WriteFile(fileType.GetPath(filename), content, 0o644)
	if err != nil {
		U.HandleErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
