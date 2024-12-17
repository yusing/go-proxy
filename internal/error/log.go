package err

import (
	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/logging"
)

func getLogger(logger ...*zerolog.Logger) *zerolog.Logger {
	if len(logger) > 0 {
		return logger[0]
	}
	return logging.GetLogger()
}

//go:inline
func LogFatal(msg string, err error, logger ...*zerolog.Logger) {
	getLogger(logger...).Fatal().Msg(err.Error())
}

//go:inline
func LogError(msg string, err error, logger ...*zerolog.Logger) {
	getLogger(logger...).Error().Msg(err.Error())
}

//go:inline
func LogWarn(msg string, err error, logger ...*zerolog.Logger) {
	getLogger(logger...).Warn().Msg(err.Error())
}

//go:inline
func LogPanic(msg string, err error, logger ...*zerolog.Logger) {
	getLogger(logger...).Panic().Msg(err.Error())
}

//go:inline
func LogInfo(msg string, err error, logger ...*zerolog.Logger) {
	getLogger(logger...).Info().Msg(err.Error())
}

//go:inline
func LogDebug(msg string, err error, logger ...*zerolog.Logger) {
	getLogger(logger...).Debug().Msg(err.Error())
}
