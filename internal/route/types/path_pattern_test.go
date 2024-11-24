package types

import (
	"errors"
	"testing"

	U "github.com/yusing/go-proxy/internal/utils/testing"
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
		_, err := ValidatePathPattern(pattern)
		U.ExpectNoError(t, err)
	}
	for _, pattern := range invalidPatterns {
		_, err := ValidatePathPattern(pattern)
		U.ExpectTrue(t, errors.Is(err, ErrInvalidPathPattern))
	}
}
