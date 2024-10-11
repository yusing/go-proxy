package errorpage

import (
	"net/http"

	. "github.com/yusing/go-proxy/internal/api/v1/utils"
)

func GetHandleFunc() http.HandlerFunc {
	setup()
	return serveHTTP
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path == "/" {
		http.Error(w, "invalid path", http.StatusNotFound)
		return
	}
	content, ok := fileContentMap.Load(r.URL.Path)
	if !ok {
		http.Error(w, "404 not found", http.StatusNotFound)
		return
	}
	if _, err := w.Write(content); err != nil {
		HandleErr(w, r, err)
	}
}
