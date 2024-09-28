package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/api/v1/error_page"
	gpHTTP "github.com/yusing/go-proxy/internal/http"
)

const staticFilePathPrefix = "/$gperrorpage/"

var CustomErrorPage = &Middleware{
	before: func(next http.Handler, w ResponseWriter, r *Request) {
		path := r.URL.Path
		if path != "" && path[0] != '/' {
			path = "/" + path
		}
		if strings.HasPrefix(path, staticFilePathPrefix) {
			filename := path[len(staticFilePathPrefix):]
			file, ok := error_page.GetStaticFile(filename)
			if !ok {
				http.NotFound(w, r)
				errPageLogger.Errorf("unable to load resource %s", filename)
			} else {
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
				w.Write(file)
			}
			return
		}
		next.ServeHTTP(w, r)
	},
	modifyResponse: func(resp *Response) error {
		// only handles non-success status code and html/plain content type
		contentType := gpHTTP.GetContentType(resp.Header)
		if !gpHTTP.IsSuccess(resp.StatusCode) && (contentType.IsHTML() || contentType.IsPlainText()) {
			errorPage, ok := error_page.GetErrorPageByStatus(resp.StatusCode)
			if ok {
				errPageLogger.Debugf("error page for status %d loaded", resp.StatusCode)
				io.Copy(io.Discard, resp.Body) // drain the original body
				resp.Body.Close()
				resp.Body = io.NopCloser(bytes.NewReader(errorPage))
				resp.ContentLength = int64(len(errorPage))
				resp.Header.Set("Content-Length", fmt.Sprint(len(errorPage)))
				resp.Header.Set("Content-Type", "text/html; charset=utf-8")
			} else {
				errPageLogger.Errorf("unable to load error page for status %d", resp.StatusCode)
			}
			return nil
		}
		return nil
	},
}

var errPageLogger = logrus.WithField("middleware", "error_page")
