package middleware

import (
	"fmt"
	"strings"
)

var middlewares = map[string]*Middleware{
	"set_x_forwarded":      SetXForwarded, // nginx
	"add_x_forwarded":      AddXForwarded, // nginx
	"trust_forward_header": AddXForwarded, // traefik alias
	"redirect_http":        RedirectHTTP,
}

func Get(name string) (middleware *Middleware, ok bool) {
	middleware, ok = middlewares[name]
	return
}

// initialize middleware names
var _ = func() (_ bool) {
	names := make(map[*Middleware][]string)
	for name, m := range middlewares {
		names[m] = append(names[m], name)
	}
	for m, names := range names {
		if len(names) > 1 {
			m.name = fmt.Sprintf("%s (a.k.a. %s)", names[0], strings.Join(names[1:], ", "))
		} else {
			m.name = names[0]
		}
	}
	return
}()
