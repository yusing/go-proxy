package middleware

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/api/v1/errorpage"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

var CustomErrorPage = &Middleware{
	before: func(next http.HandlerFunc, w ResponseWriter, r *Request) {
		if !ServeStaticErrorPageFile(w, r) {
			next(w, r)
		}
	},
	modifyResponse: func(resp *Response) error {
		// only handles non-success status code and html/plain content type
		contentType := gphttp.GetContentType(resp.Header)
		if !gphttp.IsSuccess(resp.StatusCode) && (contentType.IsHTML() || contentType.IsPlainText()) {
			errorPage, ok := errorpage.GetErrorPageByStatus(resp.StatusCode)
			if ok {
				errPageLogger.Debugf("error page for status %d loaded", resp.StatusCode)
				/* trunk-ignore(golangci-lint/errcheck) */
				io.Copy(io.Discard, resp.Body) // drain the original body
				resp.Body.Close()
				resp.Body = io.NopCloser(bytes.NewReader(errorPage))
				resp.ContentLength = int64(len(errorPage))
				resp.Header.Set("Content-Length", strconv.Itoa(len(errorPage)))
				resp.Header.Set("Content-Type", "text/html; charset=utf-8")
			} else {
				errPageLogger.Errorf("unable to load error page for status %d", resp.StatusCode)
			}
			return nil
		}
		return nil
	},
}

func ServeStaticErrorPageFile(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	if strings.HasPrefix(path, gphttp.StaticFilePathPrefix) {
		filename := path[len(gphttp.StaticFilePathPrefix):]
		file, ok := errorpage.GetStaticFile(filename)
		if !ok {
			errPageLogger.Errorf("unable to load resource %s", filename)
			return false
		}
		ext := filepath.Ext(filename)
		switch ext {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		default:
			errPageLogger.Errorf("unexpected file type %q for %s", ext, filename)
		}
		if _, err := w.Write(file); err != nil {
			errPageLogger.WithError(err).Errorf("unable to write resource %s", filename)
			http.Error(w, "Error page failure", http.StatusInternalServerError)
		}
		return true
	}
	return false
}

var errPageLogger = logrus.WithField("middleware", "error_page")
