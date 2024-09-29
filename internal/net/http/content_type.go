package http

import (
	"mime"
	"net/http"
)

type ContentType string

func GetContentType(h http.Header) ContentType {
	ct := h.Get("Content-Type")
	if ct == "" {
		return ""
	}
	ct, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return ""
	}
	return ContentType(ct)
}

func (ct ContentType) IsHTML() bool {
	return ct == "text/html" || ct == "application/xhtml+xml"
}

func (ct ContentType) IsJSON() bool {
	return ct == "application/json"
}

func (ct ContentType) IsPlainText() bool {
	return ct == "text/plain"
}
