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

	ErrExpectNoArg   = ErrInvalidArguments.Withf("expect no arg")
	ErrExpectOneArg  = ErrInvalidArguments.Withf("expect 1 arg")
	ErrExpectTwoArgs = ErrInvalidArguments.Withf("expect 2 args")
)
