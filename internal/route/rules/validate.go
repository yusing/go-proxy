package rules

import (
	"fmt"
	"os"
	"path"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/net/types"
)

type (
	ValidateFunc      func(args []string) (any, E.Error)
	Tuple[T1, T2 any] struct {
		First  T1
		Second T2
	}
	StrTuple = Tuple[string, string]
)

func (t *Tuple[T1, T2]) Unpack() (T1, T2) {
	return t.First, t.Second
}

func (t *Tuple[T1, T2]) String() string {
	return fmt.Sprintf("%v:%v", t.First, t.Second)
}

// toStrTuple returns *StrTuple.
func toStrTuple(args []string) (any, E.Error) {
	if len(args) != 2 {
		return nil, ErrExpectTwoArgs
	}
	return &StrTuple{args[0], args[1]}, nil
}

// toKVOptionalV returns *StrTuple that value is optional.
func toKVOptionalV(args []string) (any, E.Error) {
	switch len(args) {
	case 1:
		return &StrTuple{args[0], ""}, nil
	case 2:
		return &StrTuple{args[0], args[1]}, nil
	default:
		return nil, ErrExpectKVOptionalV
	}
}

// validateURL returns types.URL with the URL validated.
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

// validateAbsoluteURL returns types.URL with the URL validated.
func validateAbsoluteURL(args []string) (any, E.Error) {
	if len(args) != 1 {
		return nil, ErrExpectOneArg
	}
	u, err := types.ParseURL(args[0])
	if err != nil {
		return nil, ErrInvalidArguments.With(err)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	if u.Host == "" {
		return nil, ErrInvalidArguments.Withf("missing host")
	}
	return u, nil
}

// validateCIDR returns types.CIDR with the CIDR validated.
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

// validateURLPath returns string with the path validated.
func validateURLPath(args []string) (any, E.Error) {
	if len(args) != 1 {
		return nil, ErrExpectOneArg
	}
	p := args[0]
	trailingSlash := len(p) > 1 && p[len(p)-1] == '/'
	p, _, _ = strings.Cut(p, "#")
	p = path.Clean(p)
	if len(p) == 0 {
		return nil, ErrInvalidArguments.Withf("empty path")
	}
	if trailingSlash {
		p += "/"
	}
	return p, nil
}

// validateURLPaths returns []string with each element validated.
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

// validateFSPath returns string with the path validated.
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

// validateMethod returns string with the method validated.
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

// validateUserBCryptPassword returns *HashedCrendential with the password validated.
func validateUserBCryptPassword(args []string) (any, E.Error) {
	if len(args) != 2 {
		return nil, ErrExpectTwoArgs
	}
	return BCryptCrendentials(args[0], []byte(args[1])), nil
}

// validateModField returns CommandHandler with the field validated.
func validateModField(mod FieldModifier, args []string) (CommandHandler, E.Error) {
	setField, ok := modFields[args[0]]
	if !ok {
		return nil, ErrInvalidSetTarget.Subject(args[0])
	}
	validArgs, err := setField.validate(args[1:])
	if err != nil {
		return nil, err.Withf(setField.help.String())
	}
	modder := setField.builder(validArgs)
	switch mod {
	case ModFieldAdd:
		return modder.add, nil
	case ModFieldRemove:
		return modder.remove, nil
	}
	return modder.set, nil
}
