package middleware

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

var allMiddlewares map[string]*Middleware

var (
	ErrUnknownMiddleware    = E.New("unknown middleware")
	ErrDuplicatedMiddleware = E.New("duplicated middleware")
)

func Get(name string) (*Middleware, Error) {
	middleware, ok := allMiddlewares[strutils.ToLowerNoSnake(name)]
	if !ok {
		return nil, ErrUnknownMiddleware.
			Subject(name).
			Withf(strutils.DoYouMean(utils.NearestField(name, allMiddlewares)))
	}
	return middleware, nil
}

func All() map[string]*Middleware {
	return allMiddlewares
}

// initialize middleware names and label parsers.
func init() {
	allMiddlewares = map[string]*Middleware{
		"setxforwarded":    SetXForwarded,
		"hidexforwarded":   HideXForwarded,
		"redirecthttp":     RedirectHTTP,
		"modifyresponse":   ModifyResponse.m,
		"modifyrequest":    ModifyRequest.m,
		"errorpage":        CustomErrorPage,
		"customerrorpage":  CustomErrorPage,
		"realip":           RealIP.m,
		"cloudflarerealip": CloudflareRealIP.m,
		"cidrwhitelist":    CIDRWhiteList.m,

		// !experimental
		"forwardauth": ForwardAuth.m,
		// "oauth2":      OAuth2.m,
	}
	names := make(map[*Middleware][]string)
	for name, m := range allMiddlewares {
		names[m] = append(names[m], http.CanonicalHeaderKey(name))
	}
	for m, names := range names {
		if len(names) > 1 {
			m.name = fmt.Sprintf("%s (a.k.a. %s)", names[0], strings.Join(names[1:], ", "))
		} else {
			m.name = names[0]
		}
	}
}

func LoadComposeFiles() {
	errs := E.NewBuilder("middleware compile errors")
	middlewareDefs, err := utils.ListFiles(common.MiddlewareComposeBasePath, 0)
	if err != nil {
		logger.Err(err).Msg("failed to list middleware definitions")
		return
	}
	for _, defFile := range middlewareDefs {
		mws := BuildMiddlewaresFromComposeFile(defFile, errs)
		if len(mws) == 0 {
			continue
		}
		for name, m := range mws {
			if _, ok := allMiddlewares[name]; ok {
				errs.Add(ErrDuplicatedMiddleware.Subject(name))
				continue
			}
			allMiddlewares[strutils.ToLowerNoSnake(name)] = m
			logger.Info().
				Str("name", name).
				Str("src", path.Base(defFile)).
				Msg("middleware loaded")
		}
	}
	if errs.HasError() {
		E.LogError(errs.About(), errs.Error(), &logger)
	}
}
