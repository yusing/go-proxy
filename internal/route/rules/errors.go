package rules

import E "github.com/yusing/go-proxy/internal/error"

var (
	ErrUnterminatedQuotes     = E.New("unterminated quotes")
	ErrUnsupportedEscapeChar  = E.New("unsupported escape char")
	ErrUnknownDirective       = E.New("unknown directive")
	ErrInvalidArguments       = E.New("invalid arguments")
	ErrInvalidOnTarget        = E.New("invalid `rule.on` target")
	ErrInvalidCommandSequence = E.New("invalid command sequence")
	ErrInvalidSetTarget       = E.New("invalid `rule.set` target")

	ErrExpectNoArg       = E.Wrap(ErrInvalidArguments, "expect no arg")
	ErrExpectOneArg      = E.Wrap(ErrInvalidArguments, "expect 1 arg")
	ErrExpectTwoArgs     = E.Wrap(ErrInvalidArguments, "expect 2 args")
	ErrExpectKVOptionalV = E.Wrap(ErrInvalidArguments, "expect 'key' or 'key value'")
)
