package middleware

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	D "github.com/yusing/go-proxy/internal/docker"
)

var middlewares map[string]*Middleware

func Get(name string) (middleware *Middleware, ok bool) {
	middleware, ok = middlewares[name]
	return
}

// initialize middleware names and label parsers
func init() {
	middlewares = map[string]*Middleware{
		"set_x_forwarded":    SetXForwarded,
		"hide_x_forwarded":   HideXForwarded,
		"redirect_http":      RedirectHTTP,
		"forward_auth":       ForwardAuth.m,
		"modify_response":    ModifyResponse.m,
		"modify_request":     ModifyRequest.m,
		"error_page":         CustomErrorPage,
		"custom_error_page":  CustomErrorPage,
		"real_ip":            RealIP.m,
		"cloudflare_real_ip": CloudflareRealIP.m,
	}
	names := make(map[*Middleware][]string)
	for name, m := range middlewares {
		names[m] = append(names[m], name)
		// register middleware name to docker label parsr
		// in order to parse middleware_name.option=value into correct type
		if m.labelParserMap != nil {
			D.RegisterNamespace(name, m.labelParserMap)
		}
	}
	for m, names := range names {
		if len(names) > 1 {
			m.name = fmt.Sprintf("%s (a.k.a. %s)", names[0], strings.Join(names[1:], ", "))
		} else {
			m.name = names[0]
		}
	}
	// TODO: seperate from init()
	// b := E.NewBuilder("failed to load middlewares")
	// middlewareDefs, err := U.ListFiles(common.MiddlewareDefsBasePath, 0)
	// if err != nil {
	// 	logrus.Errorf("failed to list middleware definitions: %s", err)
	// 	return
	// }
	// for _, defFile := range middlewareDefs {
	// 	mws, err := BuildMiddlewaresFromYAML(defFile)
	// 	for name, m := range mws {
	// 		if _, ok := middlewares[name]; ok {
	// 			b.Add(E.Duplicated("middleware", name))
	// 			continue
	// 		}
	// 		middlewares[name] = m
	// 		logger.Infof("middleware %s loaded from %s", name, path.Base(defFile))
	// 	}
	// 	b.Add(err.Subject(defFile))
	// }
	// if b.HasError() {
	// 	logger.Error(b.Build())
	// }
}

var logger = logrus.WithField("module", "middlewares")
