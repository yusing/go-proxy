package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog"
)

func HandleError(logger *zerolog.Logger, err error) {
	switch {
	case err == nil, errors.Is(err, http.ErrServerClosed), errors.Is(err, context.Canceled):
		return
	default:
		logger.Fatal().Err(err).Msg("server error")
	}
}
