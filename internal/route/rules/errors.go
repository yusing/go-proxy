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

	ErrExpectNoArg       = E.New("expect no arg")
	ErrExpectOneArg      = E.New("expect 1 arg")
	ErrExpectTwoArgs     = E.New("expect 2 args")
	ErrExpectKVOptionalV = E.New("expect 'key' or 'key value'")
)
