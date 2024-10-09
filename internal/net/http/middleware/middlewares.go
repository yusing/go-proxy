package middleware

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	U "github.com/yusing/go-proxy/internal/utils"
)

var middlewares map[string]*Middleware

func Get(name string) (middleware *Middleware, ok bool) {
	middleware, ok = middlewares[U.ToLowerNoSnake(name)]
	return
}

func All() map[string]*Middleware {
	return middlewares
}

// initialize middleware names and label parsers
func init() {
	middlewares = map[string]*Middleware{
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
		"oauth2":      OAuth2.m,
	}
	names := make(map[*Middleware][]string)
	for name, m := range middlewares {
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
	b := E.NewBuilder("failed to load middlewares")
	middlewareDefs, err := U.ListFiles(common.MiddlewareComposeBasePath, 0)
	if err != nil {
		logrus.Errorf("failed to list middleware definitions: %s", err)
		return
	}
	for _, defFile := range middlewareDefs {
		mws, err := BuildMiddlewaresFromComposeFile(defFile)
		for name, m := range mws {
			if _, ok := middlewares[name]; ok {
				b.Add(E.Duplicated("middleware", name))
				continue
			}
			middlewares[U.ToLowerNoSnake(name)] = m
			logger.Infof("middleware %s loaded from %s", name, path.Base(defFile))
		}
		b.Add(err.Subject(path.Base(defFile)))
	}
	if b.HasError() {
		logger.Error(b.Build())
	}
}

var logger = logrus.WithField("module", "middlewares")
