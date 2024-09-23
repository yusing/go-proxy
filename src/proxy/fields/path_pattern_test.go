package fields

import (
	"testing"

	E "github.com/yusing/go-proxy/error"
	U "github.com/yusing/go-proxy/utils/testing"
)

var validPatterns = []string{
	"/",
	"/index.html",
	"/somepage/",
	"/drive/abc.mp4",
	"/{$}",
	"/some-page/{$}",
	"GET /",
	"GET /static/{$}",
	"GET /drive/abc.mp4",
	"GET /drive/abc.mp4/{$}",
	"POST /auth",
	"DELETE /user/",
	"PUT /storage/id/",
}

var invalidPatterns = []string{
	"/$",
	"/{$}{$}",
	"/{$}/{$}",
	"/index.html$",
	"get /",
	"GET/",
	"GET /$",
	"GET /drive/{$}/abc.mp4/",
	"OPTION /config/{$}/abc.conf/{$}",
}

func TestPathPatternRegex(t *testing.T) {
	for _, pattern := range validPatterns {
		_, err := NewPathPattern(pattern)
		U.ExpectNoError(t, err.Error())
	}
	for _, pattern := range invalidPatterns {
		_, err := NewPathPattern(pattern)
		U.ExpectError2(t, pattern, E.ErrInvalid, err.Error())
	}
}
