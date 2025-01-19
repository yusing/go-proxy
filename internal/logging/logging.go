//nolint:zerologlint
package logging

import (
	"io"
	"strings"

	"github.com/rs/zerolog"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

var logger zerolog.Logger

func InitLogger(out io.Writer) {
	var timeFmt string
	var level zerolog.Level
	var exclude []string

	switch {
	case common.IsTrace:
		timeFmt = "04:05"
		level = zerolog.TraceLevel
	case common.IsDebug:
		timeFmt = "01-02 15:04"
		level = zerolog.DebugLevel
	default:
		timeFmt = "01-02 15:04"
		level = zerolog.InfoLevel
		exclude = []string{"module"}
	}

	prefixLength := len(timeFmt) + 5 // level takes 3 + 2 spaces
	prefix := strings.Repeat(" ", prefixLength)

	logger = zerolog.New(
		zerolog.ConsoleWriter{
			Out:           out,
			TimeFormat:    timeFmt,
			FieldsExclude: exclude,
			FormatMessage: func(msgI interface{}) string { // pad spaces for each line
				msg := msgI.(string)
				lines := strutils.SplitRune(msg, '\n')
				if len(lines) == 1 {
					return msg
				}
				for i := 1; i < len(lines); i++ {
					lines[i] = prefix + lines[i]
				}
				return strutils.JoinRune(lines, '\n')
			},
		},
	).Level(level).With().Timestamp().Logger()
}

func DiscardLogger() { zerolog.SetGlobalLevel(zerolog.Disabled) }

func AddHook(h zerolog.Hook) { logger = logger.Hook(h) }

func GetLogger() *zerolog.Logger { return &logger }
func With() zerolog.Context      { return logger.With() }

func WithLevel(level zerolog.Level) *zerolog.Event { return logger.WithLevel(level) }

func Info() *zerolog.Event         { return logger.Info() }
func Warn() *zerolog.Event         { return logger.Warn() }
func Error() *zerolog.Event        { return logger.Error() }
func Err(err error) *zerolog.Event { return logger.Err(err) }
func Debug() *zerolog.Event        { return logger.Debug() }
func Fatal() *zerolog.Event        { return logger.Fatal() }
func Panic() *zerolog.Event        { return logger.Panic() }
func Trace() *zerolog.Event        { return logger.Trace() }
