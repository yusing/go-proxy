package middleware

import (
	"fmt"
	"strings"

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
		"set_x_forwarded":   SetXForwarded,
		"add_x_forwarded":   AddXForwarded,
		"redirect_http":     RedirectHTTP,
		"forward_auth":      ForwardAuth.m,
		"modify_response":   ModifyResponse.m,
		"modify_request":    ModifyRequest.m,
		"error_page":        CustomErrorPage,
		"custom_error_page": CustomErrorPage,
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
}
