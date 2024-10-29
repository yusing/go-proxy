package utils

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/logging"
)

func reqLogger(r *http.Request, level zerolog.Level) *zerolog.Event {
	return logging.WithLevel(level).Str("module", "api").
		Str("method", r.Method).
		Str("path", r.RequestURI)
}

func LogError(r *http.Request) *zerolog.Event { return reqLogger(r, zerolog.ErrorLevel) }
func LogWarn(r *http.Request) *zerolog.Event  { return reqLogger(r, zerolog.WarnLevel) }
func LogInfo(r *http.Request) *zerolog.Event  { return reqLogger(r, zerolog.InfoLevel) }
