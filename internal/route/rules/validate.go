package rules

import (
	"os"
	"path"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/types"
)

type (
	ValidateFunc func(args []string) (any, E.Error)
	StrTuple     struct {
		First, Second string
	}
)

func toStrTuple(args []string) (any, E.Error) {
	if len(args) != 2 {
		return nil, ErrExpectTwoArgs
	}
	return StrTuple{args[0], args[1]}, nil
}

func validateURL(args []string) (any, E.Error) {
	if len(args) != 1 {
		return nil, ErrExpectOneArg
	}
	u, err := types.ParseURL(args[0])
	if err != nil {
		return nil, ErrInvalidArguments.With(err)
	}
	return u, nil
}

func validateCIDR(args []string) (any, E.Error) {
	if len(args) != 1 {
		return nil, ErrExpectOneArg
	}
	if !strings.Contains(args[0], "/") {
		args[0] += "/32"
	}
	cidr, err := types.ParseCIDR(args[0])
	if err != nil {
		return nil, ErrInvalidArguments.With(err)
	}
	return cidr, nil
}

func validateURLPath(args []string) (any, E.Error) {
	if len(args) != 1 {
		return nil, ErrExpectOneArg
	}
	p := args[0]
	p, _, _ = strings.Cut(p, "#")
	p = path.Clean(p)
	if len(p) == 0 {
		return "/", nil
	}
	if p[0] != '/' {
		return nil, ErrInvalidArguments.Withf("must start with /")
	}
	return p, nil
}

func validateURLPaths(paths []string) (any, E.Error) {
	errs := E.NewBuilder("invalid url paths")
	for i, p := range paths {
		val, err := validateURLPath([]string{p})
		if err != nil {
			errs.Add(err.Subject(p))
			continue
		}
		paths[i] = val.(string)
	}
	if err := errs.Error(); err != nil {
		return nil, err
	}
	return paths, nil
}

func validateFSPath(args []string) (any, E.Error) {
	if len(args) != 1 {
		return nil, ErrExpectOneArg
	}
	p := path.Clean(args[0])
	if _, err := os.Stat(p); err != nil {
		return nil, ErrInvalidArguments.With(err)
	}
	return p, nil
}

func validateMethod(args []string) (any, E.Error) {
	if len(args) != 1 {
		return nil, ErrExpectOneArg
	}
	method := strings.ToUpper(args[0])
	if !gphttp.IsMethodValid(method) {
		return nil, ErrInvalidArguments.Subject(method)
	}
	return method, nil
}
